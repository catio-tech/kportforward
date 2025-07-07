class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.3.2"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.3.2/kportforward-darwin-arm64"
      sha256 "abbfe01af70917d4d66eefa40c6b6ce8bea98d02435e302ed0c24e202495fa06"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.3.2/kportforward-darwin-amd64"
      sha256 "40bb1f346b34804cac18adf5a5910736c5f1eb60fc02bdb0d43b98fc9df0b8e1"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.3.2/kportforward-linux-amd64"
    sha256 "76290a28ef6e95aa3517f24a07617a2092d41bfc7d4041233b18e6542b960b9e"
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
