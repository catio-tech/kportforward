class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.6"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.6/kportforward-darwin-arm64"
      sha256 "f60fc59ae592bedf03a4cbe26b38bceb523cc25f54e9e7e08b74ab2927b37aa7"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.6/kportforward-darwin-amd64"
      sha256 "2d48560f99f2c8c014f776cccbc6080fbbc74a1ab349ea80904fc1c3bff75305"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.6/kportforward-linux-amd64"
    sha256 "0a332b5e35771717b300fe9fe7247a196c5f02fc53f9a8eb2972173a817a5f23"
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
