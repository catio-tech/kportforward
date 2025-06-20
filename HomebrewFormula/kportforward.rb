class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.0.0" # Update with the latest version

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-darwin-arm64"
      # sha256 will be auto-calculated by Homebrew on first install
    else
      url "https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-darwin-amd64"
      # sha256 will be auto-calculated by Homebrew on first install
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-linux-amd64"
      # sha256 will be auto-calculated by Homebrew on first install
    end
  end

  depends_on "kubectl" => :recommended

  def install
    # The downloaded file is a binary with no extension
    # Rename it to kportforward and place it in bin
    bin.install Dir["kportforward-*"].first => "kportforward" if Dir["kportforward-*"].any?
    # If the file doesn't have a name pattern matching kportforward-*, assume it's just the binary
    bin.install Pathname.pwd.children.first => "kportforward" if !Dir["kportforward-*"].any?
    chmod 0755, bin/"kportforward"
  end

  test do
    # Test that the binary responds to version command
    assert_match(/kportforward/i, shell_output("#{bin}/kportforward version", 2))
  end
end