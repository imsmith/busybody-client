defmodule BusybodyClient.MixProject do
  @moduledoc false
  use Mix.Project

  def project do
    [
      app: :busybody_client,
      version: "0.1.0",
      elixir: "~> 1.20",
      start_permanent: false,
      deps: deps(),
      description: "Register a Phoenix app with a local busybody directory and heartbeat",
      package: [licenses: ["AGPL-3.0-only"], links: %{}]
    ]
  end

  # Library only: no `mod`, so consumers never start anything they did not ask for.
  def application do
    [extra_applications: [:logger, :inets, :public_key]]
  end

  defp deps do
    []
  end
end
