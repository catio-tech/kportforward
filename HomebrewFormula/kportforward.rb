class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.4"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.4/kportforward-darwin-arm64"
      sha256 "1a4723dd16bb9916c1ac163055dfea63f1b7bbc90a4e1b823ec0de74380ce9f4"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.4/kportforward-darwin-amd64"
      sha256 "98475d34b6d2f90ad5cb023a2a096292b691f625bac3bac1abf36316656f87e5"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.4/kportforward-linux-amd64"
    sha256 "4c4a6d398729e94e42b2c82f9767e8de53fd795919d97465c83e4d953e8d3d60"
  end

  depends_on "kubectl" => :recommended

  def install
    # Move the downloaded binary to the bin directory with the name "kportforward"
    # First, find what files we have in the current directory
    binary = Dir["*"].first
    bin.install binary => "kportforward"
    
    # Ensure binary is executable
    chmod 0755, bin/"kportforward"
  end

  test do
    assert_match(/kportforward/i, shell_output("#{bin}/kportforward version 2>&1", 2))
  end
end
