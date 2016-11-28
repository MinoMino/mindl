import os.path
import sys

if __name__ == "__main__":
    if len(sys.argv) < 4:
        print("Usage: {} <zip|tar> <output> <file> [file ...]"
            .format(sys.argv[0]))
        sys.exit(0)

    cmd = sys.argv[1]
    out = sys.argv[2]
    files = sys.argv[3:]
    if cmd.lower() == "zip":
        from zipfile import ZipFile, ZIP_DEFLATED
        with ZipFile(out + ".zip", "w", compression=ZIP_DEFLATED) as zf:
            for file in files:
                zf.write(file, arcname=os.path.basename(file))
    elif cmd.lower() == "tar":
        import tarfile
        with tarfile.open(out + ".tar.gz", "w:gz") as tf:
            for file in files:
                tf.add(file, arcname=os.path.basename(file))
    else:
        print("{} is not a valid format.".format(cmd))
        sys.exit(1)