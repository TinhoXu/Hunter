#!/usr/bin/bash

archs_mac=(amd64 arm64)
archs_linux=(386 amd64 arm arm64)
archs_windows=(386 amd64)

app_name=nf
app_version=V1.0

pkg_dir=releases/${app_version}

echo "编译 mac 版"
for arch in ${archs_mac[@]}; do
  CGO_ENABLED=0 GOOS=darwin GOARCH=${arch} go build -o ${pkg_dir}/${app_name}_${app_version}_darwin_${arch}
done

echo "编译 linux 版"
for arch in ${archs_linux[@]}; do
  CGO_ENABLED=0 GOOS=linux GOARCH=${arch} go build -o ${pkg_dir}/${app_name}_${app_version}_linux_${arch}
done

echo "编译 windows 版"
for arch in ${archs_windows[@]}; do
  CGO_ENABLED=0 GOOS=windows GOARCH=${arch} go build -o ${pkg_dir}/${app_name}_${app_version}_windows_${arch}.exe
done

echo "编译完成"
