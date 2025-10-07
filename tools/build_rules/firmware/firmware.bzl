"""
A repository rule to download and unpack the root filesystem from a Kindle firmware update.
"""

def _impl(rctx):
    # type: (repository_ctx) -> None
    quiet = rctx.attr.quiet  # type: bool

    rctx.report_progress("Downloading firmware image")
    rctx.download(
        url = rctx.attr.url,
        sha256 = rctx.attr.sha256,
        output = "firmware.bin",
    )

    staging_dir = rctx.path("__staging_dir__")

    rctx.report_progress("kindletool extracting")

    # TODO: stop assuming this is in PATH
    rctx.execute([
        "kindletool",
        "extract",
        rctx.path("firmware.bin"),
        staging_dir,
    ], quiet = quiet)

    rctx.report_progress("ungzipping root image")
    gzipped_rootfs = rctx.path("{}/rootfs.img.gz".format(staging_dir))
    ungzipped_rootfs = rctx.path("{}/rootfs.img".format(staging_dir))
    rctx.execute([
        "7z",
        "x",
        "-o{}".format(ungzipped_rootfs),
        gzipped_rootfs,
    ], quiet = quiet)

    rctx.report_progress("extracting ext3 filesystem")
    rctx.execute([
        "7z",
        "x",
        "-aos",
        "-snld",
        "-o.",
        ungzipped_rootfs,
    ], quiet = quiet)

    rctx.execute([
        "7z",
        "x",
        "-aos",
        "-snld",
        "-o.",
        ungzipped_rootfs,
    ], quiet = quiet)

    rctx.template(
        "BUILD.bazel",
        rctx.attr._build_file,
    )

    rctx.execute([
        "rm",
        "-rf",
        staging_dir,
    ], quiet = quiet)

kindle_firmware = repository_rule(
    implementation = _impl,
    attrs = {
        "quiet": attr.bool(default = True),
        "url": attr.string(mandatory = True),
        "sha256": attr.string(mandatory = True),
        "_build_file": attr.label(default = Label("//tools/build_rules/firmware:firmware.BUILD.bazel")),
    },
)
