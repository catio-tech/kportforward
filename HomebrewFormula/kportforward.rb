class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.3.0"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.3.0/kportforward-darwin-arm64"
      sha256 "924a7e88e31693f483fe80fd1830c797658784e3189dc64e7aa0d6c1c1396269"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.3.0/kportforward-darwin-amd64"
      sha256 "c2140a3447436f8043b88e9a4191632a021e7c805803a2c5eea0cb4f3b09405b"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.3.0/kportforward-linux-amd64"
    sha256 "436d5339f50e2fcc9933102b9690947ac8cdabdfa6ef1e7576f521853b34eb5f"
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
