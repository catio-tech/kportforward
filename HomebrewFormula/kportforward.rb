class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.6.2"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.6.2/kportforward-darwin-arm64"
      sha256 "92585fa6f4a10a46954d52decba1dbe2ead1bcfca66d70cf65e21da3b917f02a"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.6.2/kportforward-darwin-amd64"
      sha256 "85ea533a90ac727241d7f41dde02d8c2f237c170d93b398e9ab9488d6b1ba3a4"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.6.2/kportforward-linux-amd64"
    sha256 "7b15f9811094941392c44c400a3f1ebead4d0393f085695757ca3b2d409fcd36"
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
