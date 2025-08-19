class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.1"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.1/kportforward-darwin-arm64"
      sha256 "b241db65e8d3ea2a4cc5af17fcc4788a1aee1e3ffaf2a9bd6fd1aa89d7290e79"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.1/kportforward-darwin-amd64"
      sha256 "184efe7f451909a787720e06028cbbfedf8dd232d1f9d3a5be0c38b32d529fc3"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.1/kportforward-linux-amd64"
    sha256 "c23cb99646982e72e2645b6e3179d2d1831dfb65fe5807060519de9f2faa1d96"
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
