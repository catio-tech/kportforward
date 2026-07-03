class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.6.1"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.6.1/kportforward-darwin-arm64"
      sha256 "0aec7d548f0f710aaa209a5b69df482572e1505ebbf20fd8cac96779bd6cb914"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.6.1/kportforward-darwin-amd64"
      sha256 "680859ddd77c87104ed3e08e54ef07cd21c56eae0b28d701f7878cf6abce6a30"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.6.1/kportforward-linux-amd64"
    sha256 "e44a610db8c5dd98ad497bc99c852825087499ed82f3d570fa14181b8d85676e"
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
