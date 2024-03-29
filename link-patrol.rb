# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class LinkPatrol < Formula
  desc "Detect dead URLs in markdown files"
  homepage "https://github.com/rednafi/link-patrol"
  version "0.6"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rednafi/link-patrol/releases/download/0.6/link-patrol_Darwin_arm64.tar.gz"
      sha256 "3da0edd69f0f378f1e2d33c65e7aeea99a47c20b56607b7f1ba34330faf57a01"

      def install
        bin.install "link-patrol"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/rednafi/link-patrol/releases/download/0.6/link-patrol_Darwin_x86_64.tar.gz"
      sha256 "63ee41a59ffd33538d132ff72b47527ff9b74ed313ea35784f35fc7fc33cb0ae"

      def install
        bin.install "link-patrol"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/rednafi/link-patrol/releases/download/0.6/link-patrol_Linux_x86_64.tar.gz"
      sha256 "16bf2bdb4a73d351bc1afce68fcf05aa060cfbecc660dc0e43a80fd4108aecb8"

      def install
        bin.install "link-patrol"
      end
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/rednafi/link-patrol/releases/download/0.6/link-patrol_Linux_arm64.tar.gz"
      sha256 "95c0b9379d38489888ca65770727033dc2c0e6797281ce2b59fd82985133fc0e"

      def install
        bin.install "link-patrol"
      end
    end
  end
end
