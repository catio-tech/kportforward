class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.3.0"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.3.0/kportforward-darwin-arm64"
      sha256 "189b88d0b67ee39b586ab86c07d14350da0ebf085f57d1a874decfb91601bc4b"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.3.0/kportforward-darwin-amd64"
      sha256 "492273742038c9ab18ed76b7b2b45252b67ff4da2c65806118932dd531293125"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.3.0/kportforward-linux-amd64"
    sha256 "5ed8662ccaadd9196e237301a854651b5ad52c65d08952b92910cfbf9653a93d"
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
