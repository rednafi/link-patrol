# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class LinkPatrol < Formula
  desc "Detect dead URLs in markdown files"
  homepage "https://github.com/rednafi/link-patrol"
  version "0.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rednafi/link-patrol/releases/download/v0.4/link-patrol_Darwin_arm64.tar.gz"
      sha256 "9dc4e6e200404579383e7168681e2bb2d39ae6c4e909560aa2785d10e3c24739"

      def install
        bin.install "link-patrol"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/rednafi/link-patrol/releases/download/v0.4/link-patrol_Darwin_x86_64.tar.gz"
      sha256 "0015eccf06ce1f29f8f6a61698fec646470778f83fab8b83aa041c2ba27aa9b9"

      def install
        bin.install "link-patrol"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/rednafi/link-patrol/releases/download/v0.4/link-patrol_Linux_x86_64.tar.gz"
      sha256 "7e6f97fa89023a2a6efc4b38d584a45efc53e6cbf547435c123cae44346a40c6"

      def install
        bin.install "link-patrol"
      end
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/rednafi/link-patrol/releases/download/v0.4/link-patrol_Linux_arm64.tar.gz"
      sha256 "52652f358cdee53e00eeec0e81afbcb0efd8a8a63a1dd987b9a8a6f731c1952d"

      def install
        bin.install "link-patrol"
      end
    end
  end
end
