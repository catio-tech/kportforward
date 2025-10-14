class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.2"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.2/kportforward-darwin-arm64"
      sha256 "6cf99290ea4a1324d77f6dc260489f6559c861e9ba84a830f56fe543bff31802"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.2/kportforward-darwin-amd64"
      sha256 "de534de5590fa725e5679373d1562329f4bbdc357bb65b61d17dca86c8454ad6"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.2/kportforward-linux-amd64"
    sha256 "b9df8c971260249a79df7e30cc1a91a761fdfd14acdb399a296d39b557eaa33e"
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
