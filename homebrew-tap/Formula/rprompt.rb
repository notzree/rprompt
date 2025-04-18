class Rprompt < Formula
  desc "A CLI tool for managing and generating prompts"
  homepage "https://github.com/notzree/rprompt"
  version "0.1.0" # Update this with your actual version

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/notzree/rprompt/releases/download/v0.1.0/rprompt-darwin-arm64.tar.gz"
    sha256 "49017eec1d59d743a438d03b93dd1aadd22b0e567db1be63ce69ec1afba713c2"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/notzree/rprompt/releases/download/v0.1.0/rprompt-darwin-amd64.tar.gz"
    sha256 "43254109b74c41c93f05358ee27af335d648d39aef03bd52d5f6e5288d4c8647"
  end

  def install
    bin.install "rprompt"
  end

  test do
    system "#{bin}/rprompt", "--help"
  end
end
