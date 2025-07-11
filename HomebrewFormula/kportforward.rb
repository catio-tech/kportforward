class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.4.1"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.4.1/kportforward-darwin-arm64"
      sha256 "da3648b253fd4a74158c438139ad522e4a23841e01de8e81e88297476e0c482b"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.4.1/kportforward-darwin-amd64"
      sha256 "b59501fdbbf02c86b71832c5df70aad479b7f86ebf58345c81e0bc7f024963eb"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.4.1/kportforward-linux-amd64"
    sha256 "4a294e72720b93bc0f03705ab431ccebb99ad80dfdba5bd69cb23dd2222e2e89"
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
