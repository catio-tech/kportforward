class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.0.0"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-darwin-arm64"
      sha256 "b10e774fb0ec9bcf57dc5c580aad98d831501a86b03315c7d1cceef060c9fb57"
    else
      url "https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-darwin-amd64"
      sha256 "7f21f85bc915c2fc4e29e7f42fdc0bb02d3cc4af4fc8e5ac1f1ca0b4c1cee9c0"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-linux-amd64"
    sha256 "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  end

  depends_on "kubectl" => :recommended

  def install
    # The downloaded binary has no file extension
    binary_name = File.basename(url.to_s.split("/").last)
    
    # Move the binary to the bin directory with the correct name
    bin.install binary_name => "kportforward"
    
    # Ensure binary is executable
    chmod 0755, bin/"kportforward"
  end

  test do
    assert_match(/kportforward/i, shell_output("#{bin}/kportforward version 2>&1", 2))
  end
end