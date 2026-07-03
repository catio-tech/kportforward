class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.6.0"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.6.0/kportforward-darwin-arm64"
      sha256 "a23cd962ea73e862ba800b1a134e9d1850a19f3026b8b26bbb95601656867ade"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.6.0/kportforward-darwin-amd64"
      sha256 "2d1fa224899139db3d6d5934d758893428c9e01f6965dc5085b36824e0d9a187"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.6.0/kportforward-linux-amd64"
    sha256 "a6517737ca4f76bf88b7a817e5923444eac5f08f1200f1b4a6871931af5ca425"
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
