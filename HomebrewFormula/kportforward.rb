class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.7.1"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.7.1/kportforward-darwin-arm64"
      sha256 "b6bb75ac8b2740de2281f4d61421eab9101afe8286220554ef05c810c1f9e93c"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.7.1/kportforward-darwin-amd64"
      sha256 "4a82622ce3865095af676d0cf44d40499376cf13b681a8fb780e8f155dda9c89"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.7.1/kportforward-linux-amd64"
    sha256 "c023b0c161e56d73ea7e04f459cf3d644118c97857c03b5a64e4311cfb158705"
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
