class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.7"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.7/kportforward-darwin-arm64"
      sha256 "5e73ede317860d99d7e60dfa3967bbc1f4b07f5ce00946b8673a1283fd2afada"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.7/kportforward-darwin-amd64"
      sha256 "bdafd23c5baadc1f7a78222313635af6a6aa97f7e7592756bd2735e6e9b798e3"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.7/kportforward-linux-amd64"
    sha256 "e31c22ba8694662e7e6fba9780b77b60518f246b99ed50041b786b48c992f00b"
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
