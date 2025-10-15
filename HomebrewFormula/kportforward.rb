class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.5"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.5/kportforward-darwin-arm64"
      sha256 "739be11b1b38e04d5ba53e0f6450eb411cde6ae2f30139a5b057fc537c2b929d"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.5/kportforward-darwin-amd64"
      sha256 "a8d33ce29a3ce3d70bcabd85443521f497f7ce612994426b607bc62c020f934f"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.5/kportforward-linux-amd64"
    sha256 "d513ceea5c18418cbefd15d2174dcb42a3f7d34c831a5fb4b56f181859b00212"
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
