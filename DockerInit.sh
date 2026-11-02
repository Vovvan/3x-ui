#!/bin/sh
set -e
# Vovvan/Xray-core v26.10.21-mixed.1 ships linux/amd64 only. Reject other
# targets instead of publishing an image whose Mixed registry is missing.
case ${1:-amd64} in
    amd64)
        ;;
    *)
        echo "DockerInit: Vovvan/Xray-core v26.10.21-mixed.1 is linux/amd64 only (got '$1')" >&2
        exit 1
        ;;
esac
PLATFORM="amd64"
XRAY_ZIP="64"
XRAY_BIN="xray-linux-amd64"
MTG_BIN="mtg-linux-amd64"
MTG_MULTI_VER=$(curl -sfL "https://api.github.com/repos/mhsanaei/mtg-multi/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)
if [ -z "$MTG_MULTI_VER" ]; then
    echo "DockerInit: could not resolve the latest mtg-multi release tag" >&2
    exit 1
fi
mkdir -p build/bin
cd build/bin
curl -sfLRO "https://github.com/Vovvan/Xray-core/releases/download/v26.10.21-mixed.1/Xray-linux-${XRAY_ZIP}.zip"
curl -sfLRO "https://github.com/Vovvan/Xray-core/releases/download/v26.10.21-mixed.1/Xray-linux-${XRAY_ZIP}.zip.dgst"
EXPECTED_SHA=$(sed -n 's/^SHA2-256= *//p' "Xray-linux-${XRAY_ZIP}.zip.dgst" | head -n 1 | tr -d '[:space:]')
ACTUAL_SHA=$(sha256sum "Xray-linux-${XRAY_ZIP}.zip" | awk '{print $1}')
if [ -z "$EXPECTED_SHA" ] || [ "$EXPECTED_SHA" != "$ACTUAL_SHA" ]; then
    echo "DockerInit: Xray archive checksum mismatch (expected '$EXPECTED_SHA', got '$ACTUAL_SHA')" >&2
    exit 1
fi
unzip -oq "Xray-linux-${XRAY_ZIP}.zip"
rm -f "Xray-linux-${XRAY_ZIP}.zip" "Xray-linux-${XRAY_ZIP}.zip.dgst" LICENSE README.md geoip.dat geosite.dat
mv xray "${XRAY_BIN}"
# mtg-multi (MTProto sidecar) ships prebuilt amd64 release binaries, so
# download and unpack instead of compiling it here.
MTG_PKG="mtg-multi-${MTG_MULTI_VER#v}-linux-${PLATFORM}"
curl -sfLRO "https://github.com/mhsanaei/mtg-multi/releases/download/${MTG_MULTI_VER}/${MTG_PKG}.tar.gz"
tar -xzf "${MTG_PKG}.tar.gz"
mv "${MTG_PKG}/mtg-multi" "${MTG_BIN}"
rm -rf "${MTG_PKG}" "${MTG_PKG}.tar.gz"
chmod +x "${MTG_BIN}"
curl -sfLRO https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat
curl -sfLRO https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat
curl -sfLRo geoip_IR.dat https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geoip.dat
curl -sfLRo geosite_IR.dat https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geosite.dat
curl -sfLRo geoip_RU.dat https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat
curl -sfLRo geosite_RU.dat https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat
cd ../../
