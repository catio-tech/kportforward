class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.0"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.0/kportforward-darwin-arm64"
      sha256 "c652fd67139161af0c47f291049b897da46e2f7a6c5fe9297cb4dc8fe41172e0"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.0/kportforward-darwin-amd64"
      sha256 "7532c55f4c69221ea5a080f4aab0e7e3c96d558d636094df066029b063f1673a"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.0/kportforward-linux-amd64"
    sha256 "c2ffee3236286b6d1e76d0c00a9f59340d52035ddf848adeb08438eaa6538e1b"
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
