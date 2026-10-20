defmodule Busybody.Client do
  @moduledoc """
  GenServer for consumer apps to register with the Busybody directory.

  Add to your supervision tree (dev only):

      children = [
        MyAppWeb.Endpoint,
        {Busybody.Client, name: "my_app", endpoint: MyAppWeb.Endpoint}
      ]

  Options:
    * `:name` - app name to register (required)
    * `:endpoint` - Phoenix endpoint module, used to resolve port (required unless `:port` given)
    * `:port` - explicit port (overrides endpoint resolution)
    * `:registry_url` - defaults to "http://localhost:5150"
    * `:host` - defaults to system hostname
    * `:group` - optional group membership shown in the directory
    * `:heartbeat_interval` - ms between re-registrations (default 60s)
    * `:process_name` - registered name for this GenServer (default `Busybody.Client`)
  """

  use GenServer
  require Logger

  @default_registry_url "http://localhost:5150"
  @heartbeat_interval :timer.seconds(60)
  @startup_delay :timer.seconds(2)

  @typedoc "JSON body sent to `POST /api/register`."
  @type payload :: %{
          required(:name) => String.t(),
          required(:port) => pos_integer(),
          required(:pid) => String.t(),
          required(:host) => String.t(),
          optional(:group) => String.t()
        }

  @doc "Start the client GenServer. See module docs for available options."
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    GenServer.start_link(__MODULE__, opts, name: Keyword.get(opts, :process_name, __MODULE__))
  end

  @doc """
  Build the JSON registration payload from the given opts and bound port.

  ## Examples

      iex> payload = Busybody.Client.build_payload([name: "demo", host: "devbox"], 4000)
      iex> payload.name
      "demo"
      iex> payload.port
      4000
      iex> payload.host
      "devbox"
      iex> is_binary(payload.pid)
      true

  """
  @spec build_payload(keyword(), pos_integer()) :: payload()
  def build_payload(opts, port) do
    {:ok, hostname} = :inet.gethostname()

    %{
      name: Keyword.fetch!(opts, :name),
      port: port,
      pid: System.pid() |> to_string(),
      host: Keyword.get(opts, :host, to_string(hostname))
    }
    |> maybe_put(:group, Keyword.get(opts, :group))
  end

  defp maybe_put(map, _key, nil), do: map
  defp maybe_put(map, key, value), do: Map.put(map, key, value)

  @doc """
  Resolve the port from opts: an explicit `:port`, or the port the
  `:endpoint` actually bound. Returns `{:error, :no_server}` when the
  endpoint is running without a server (`mix run`, tasks, tests), so the
  client can wait instead of crashing.

  ## Examples

      iex> Busybody.Client.resolve_port(port: 4567)
      {:ok, 4567}

      iex> Busybody.Client.resolve_port(name: "oops")
      ** (ArgumentError) Busybody.Client requires either :port or :endpoint option

  """
  @spec resolve_port(keyword()) :: {:ok, pos_integer()} | {:error, term()}
  def resolve_port(opts) do
    cond do
      Keyword.has_key?(opts, :port) ->
        {:ok, Keyword.fetch!(opts, :port)}

      Keyword.has_key?(opts, :endpoint) ->
        case Keyword.fetch!(opts, :endpoint).server_info(:http) do
          {:ok, {_ip, port}} -> {:ok, port}
          {:error, reason} -> {:error, reason}
        end

      true ->
        raise ArgumentError, "Busybody.Client requires either :port or :endpoint option"
    end
  end

  @impl true
  def init(opts) do
    state = %{
      opts: opts,
      port: 0,
      registry_url: Keyword.get(opts, :registry_url, @default_registry_url),
      heartbeat_interval: Keyword.get(opts, :heartbeat_interval, @heartbeat_interval)
    }

    Process.send_after(self(), :heartbeat, @startup_delay)
    {:ok, state}
  end

  # Port 0 means "not bound yet". Each tick retries resolution until the
  # endpoint reports a bound port; an error is ignored, not raised.
  @impl true
  def handle_info(:heartbeat, state) do
    state =
      case resolve_port(state.opts) do
        {:ok, port} when port > 0 ->
          do_register(state.registry_url, build_payload(state.opts, port))
          %{state | port: port}

        _ ->
          state
      end

    schedule_heartbeat(state.heartbeat_interval)
    {:noreply, state}
  end

  defp do_register(registry_url, payload) do
    url = "#{registry_url}/api/register"
    body = JSON.encode!(payload)

    case :httpc.request(
           :post,
           {to_charlist(url), [], ~c"application/json", to_charlist(body)},
           [timeout: 5000],
           []
         ) do
      {:ok, {{_, 200, _}, _, _}} ->
        Logger.debug("Busybody: registered #{payload.name} on port #{payload.port}")

      {:error, reason} ->
        Logger.warning("Busybody: registration failed: #{inspect(reason)}")

      other ->
        Logger.warning("Busybody: unexpected response: #{inspect(other)}")
    end
  end

  defp schedule_heartbeat(interval) do
    Process.send_after(self(), :heartbeat, interval)
  end
end
