class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.3"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.3/kportforward-darwin-arm64"
      sha256 "b127036c8578cbfe1917064d784f314fa553610157b221f8175fc9bb8166d3ff"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.3/kportforward-darwin-amd64"
      sha256 "4f99e8c3621367d67cc9568e03165bfd6dad4d9c1a0a4621fac0011d934657cb"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.3/kportforward-linux-amd64"
    sha256 "71083df9017bc01ff2c9aede5f7c8d2196aa59c37f67bc85860e678ff249edb0"
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
