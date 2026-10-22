defmodule Busybody.ClientTest do
  @moduledoc false
  use ExUnit.Case, async: true

  alias Busybody.Client

  describe "build_payload/1" do
    test "builds registration payload from opts" do
      payload = Client.build_payload([name: "myapp", host: "devbox"], 4000)

      assert payload.name == "myapp"
      assert payload.port == 4000
      assert payload.host == "devbox"
      assert is_binary(payload.pid)
    end

    test "defaults host to system hostname" do
      payload = Client.build_payload([name: "myapp"], 4000)
      {:ok, hostname} = :inet.gethostname()
      assert payload.host == to_string(hostname)
    end

    test "omits group unless given" do
      refute Map.has_key?(Client.build_payload([name: "myapp"], 4000), :group)
      assert Client.build_payload([name: "myapp", group: "infra"], 4000).group == "infra"
    end

    test "uses system pid" do
      payload = Client.build_payload([name: "myapp"], 4000)
      assert payload.pid == System.pid() |> to_string()
    end
  end

  describe "resolve_port/1" do
    test "returns port from explicit opt" do
      assert Client.resolve_port(port: 4567) == {:ok, 4567}
    end

    test "returns error, not a crash, when the endpoint has no server" do
      defmodule NoServer do
        def server_info(:http), do: {:error, :no_server_found}
      end

      assert Client.resolve_port(endpoint: NoServer) == {:error, :no_server_found}
    end

    test "warns once when the endpoint has a fixed port" do
      defmodule Fixed do
        def server_info(:http), do: {:ok, {{127, 0, 0, 1}, 4000}}
        def config(:http), do: [port: 4000]
      end

      defmodule Random do
        def server_info(:http), do: {:ok, {{0, 0, 0, 0}, 41631}}
        def config(:http), do: [port: 0]
      end

      log =
        ExUnit.CaptureLog.capture_log(fn ->
          {:ok, pid} =
            Client.start_link(
              name: "fixed",
              endpoint: Fixed,
              process_name: :fixed_client,
              registry_url: "http://127.0.0.1:9"
            )

          send(pid, :heartbeat)
          send(pid, :heartbeat)
          :sys.get_state(pid)
          GenServer.stop(pid)
        end)

      assert [_] = Regex.scan(~r/binds fixed port 4000/, log)

      log =
        ExUnit.CaptureLog.capture_log(fn ->
          {:ok, pid} =
            Client.start_link(
              name: "random",
              endpoint: Random,
              process_name: :random_client,
              registry_url: "http://127.0.0.1:9"
            )

          send(pid, :heartbeat)
          :sys.get_state(pid)
          GenServer.stop(pid)
        end)

      refute log =~ "fixed port"
    end

    test "raises when neither port nor endpoint given" do
      assert_raise ArgumentError, ~r/requires either/, fn ->
        Client.resolve_port(name: "myapp")
      end
    end
  end
end
