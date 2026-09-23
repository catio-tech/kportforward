class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.7.0"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.7.0/kportforward-darwin-arm64"
      sha256 "f844df1ecda8c26714d787790a03eac4c56e013f750dda37829f13d95603f6e9"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.7.0/kportforward-darwin-amd64"
      sha256 "0412f1fdbd56e49d47d9b8d82d7d1b47fdd957c87ffa4c5039d3c51693f759c8"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.7.0/kportforward-linux-amd64"
    sha256 "4ad235968a396ec7e365245d207cd0df85ed1673c2bc5842781d33deac65f070"
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
