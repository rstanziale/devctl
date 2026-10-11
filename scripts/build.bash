#!/usr/bin/env bash

# HOW TO USE THIS SCRIPT
#
# Purpose:
# Build devctl for multiple target platforms and package each binary in an archive.
#
# Prerequisites:
# - Bash environment
# - Go installed and available in PATH
# - For Windows archive creation, one of these options:
#   - zip command available, or
#   - powershell.exe and cygpath available (Git Bash fallback)
#
# Usage:
#   bash scripts/build.bash <go-package>
#
# Example:
#   bash scripts/build.bash github.com/rs-box/devctl
#
# Output:
# - Artifacts are generated in the ./bin directory.
# - Windows target: <package-name>-windows-amd64.zip
# - Linux target:   <package-name>-linux-amd64.tar.gz
#
# Each archive contains:
# - The devctl binary
# - README.md
# - configs/config.example.yaml
# - configs/config.schema.json

package=$1 

if [[ -z "$package" ]]; then
  echo "usage: $0 <package-name>"
  exit 1
fi

package_split=(${package//\// })
package_name=${package_split[-1]}
    
platforms=("windows/amd64" "linux/amd64")

mkdir -p bin

bin_dir_abs="$(pwd)/bin"

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}

    if [ "$GOOS" = "windows" ]; then
        executable_name='devctl.exe'
        archive_name=$package_name'-'$GOOS'-'$GOARCH'.zip'
    else
        executable_name='devctl'
        archive_name=$package_name'-'$GOOS'-'$GOARCH'.tar.gz'
    fi

    work_dir=$(mktemp -d)
    if [ $? -ne 0 ]; then
        echo 'Unable to create temporary build directory. Aborting...'
        exit 1
    fi

    env GOOS=$GOOS GOARCH=$GOARCH go build -o "$work_dir/$executable_name" $package
    if [ $? -ne 0 ]; then
        rm -rf "$work_dir"
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi

    # Copy distribution files alongside the binary
    cp README.md "$work_dir/README.md"
    mkdir -p "$work_dir/configs"
    cp configs/config.example.yaml "$work_dir/configs/config.example.yaml"
    cp configs/config.schema.json "$work_dir/configs/config.schema.json"

    archive_path="$bin_dir_abs/$archive_name"

    if [ "$GOOS" = "windows" ]; then
        if command -v zip >/dev/null 2>&1; then
            (
                cd "$work_dir" || exit 1
                zip -q -r "$archive_path" .
            )
            if [ $? -ne 0 ]; then
                rm -rf "$work_dir"
                echo 'Unable to create zip archive with zip command. Aborting...'
                exit 1
            fi
        elif command -v powershell.exe >/dev/null 2>&1 && command -v cygpath >/dev/null 2>&1; then
            work_dir_win_path=$(cygpath -w "$work_dir")
            archive_win_path=$(cygpath -w "$archive_path")

            powershell.exe -NoProfile -Command "Compress-Archive -Path '$work_dir_win_path\*' -DestinationPath '$archive_win_path' -Force" >/dev/null
            if [ $? -ne 0 ]; then
                rm -rf "$work_dir"
                echo 'Unable to create zip archive with PowerShell fallback. Aborting...'
                exit 1
            fi
        else
            rm -rf "$work_dir"
            echo 'Unable to create zip archive: install zip, or run in Git Bash with powershell.exe and cygpath available.'
            exit 1
        fi
    else
        tar -C "$work_dir" -czf "$archive_path" .
        if [ $? -ne 0 ]; then
            rm -rf "$work_dir"
            echo 'Unable to create tar.gz archive. Aborting...'
            exit 1
        fi
    fi

    rm -rf "$work_dir"
done
