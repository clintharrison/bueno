_DEFAULT_MTIME = 946699200

def ext_dir(name):
    """
    Creates an mtree entry for a directory.
    """
    name = "./" + name.lstrip("./")
    return "{} uid=0 gid=0 mode=0755 type=dir time={}".format(name, _DEFAULT_MTIME)

def ext_file(name, src):
    """
    Creates an mtree entry for a file.
    """
    name = "./" + name.lstrip("./")
    return "{} uid=0 gid=0 mode=0755 type=file content={} time={}".format(name, src, _DEFAULT_MTIME)
