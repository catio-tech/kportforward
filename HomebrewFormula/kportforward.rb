class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.5"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.5/kportforward-darwin-arm64"
      sha256 "a696307a245ab0bb5d66c48106f4dc28c5ea1ba0471ecc20b0b005c756c9cd1b"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.5/kportforward-darwin-amd64"
      sha256 "4e635fa2a869bf6322dbe99d2735fd94a4a505b0e30c9cf10fd4bc3b03ba23be"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.5/kportforward-linux-amd64"
    sha256 "c9322274ab990df64e9f1876f68f7600b547ab9544af126bdaae8f83c8c052c7"
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
