class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  version "1.5.8"
  license "MIT"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.8/kportforward-darwin-arm64"
      sha256 "8920b6805d97b10c1f2cefea0cea1dfd474d489c5b2fdda5f36c21acd1531af4"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.8/kportforward-darwin-amd64"
      sha256 "82ee0eeef00b09ce03a9e3cb2409000f655e5f33372468f9fbee740fcea1cf47"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.8/kportforward-linux-amd64"
    sha256 "229c9f20917146a8296a24bbfa7fec58142a009a5ca23d496a8a18517b8cf7c3"
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
