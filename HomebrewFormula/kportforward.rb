class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.8"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.8/kportforward-darwin-arm64"
      sha256 "4dea431bcbb4590cc52740aa1caa3f8eac3c647427acb4d06965e3f38a654cb0"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.8/kportforward-darwin-amd64"
      sha256 "6ec0cd6e4c04177c396740e7f044107efe508a9d5622dccbd0482d87b4d47f5b"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.8/kportforward-linux-amd64"
    sha256 "fdec2c0745d855ffb29f8e82306674d76b972d1431cb963e5cdd20653dbf3147"
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
