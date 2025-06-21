class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.3.1"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.3.1/kportforward-darwin-arm64"
      sha256 "e27d7ad8e6b51388ea247810353edc8c192e5bf054c42a4594a84c56af689ae2"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.3.1/kportforward-darwin-amd64"
      sha256 "8d53a9a53ce1ae97040138277ee1933bd0c47875cef682d2f8973346084a27c4"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.3.1/kportforward-linux-amd64"
    sha256 "4e416fbb3e701dfd568286bb3e56961844398acd070e689185f645d571794860"
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
