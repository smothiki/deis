import os.path
import shutil
import subprocess
import tempfile

from django.conf import settings


def publish_release(src_image, config, release_image):
    """
    Publish a new release as a Docker image

    Given a source repository path, a dictionary of environment variables
    and a target tag, create a new lightweight Docker image on the registry.

    For example publish_release('gabrtv/myapp:<sha>', {'ENVVAR': 'values'}, 'v23') 
    results in a new Docker image at: <registry_url>/gabrtv/myapp:v23
    which contains the new configuration as ENV entries.
    """
    # prepare image names
    target_image = '{}:{}/{}'.format(settings.REGISTRY_HOST, settings.REGISTRY_PORT, release_image)
    # write out dockerfile
    dockerfile = _build_dockerfile(src_image, config)
    tempdir = tempfile.mkdtemp()
    dockerfile_path = os.path.join(tempdir, 'Dockerfile')
    with open(dockerfile_path, 'w') as f:
        f.write(dockerfile)
    try:
        # build the new image with last-mile configuration
        p = subprocess.Popen(['docker', 'build', '-t', release_image, tempdir])
        rc = p.wait()
        if rc != 0:
            raise RuntimeError('Failed to build release image')
        # tag the release image
        p = subprocess.Popen(['docker', 'tag', release_image, target_image])
        rc = p.wait()
        if rc != 0:
            raise RuntimeError('Failed to tag release image')
        # push the release image
        p = subprocess.Popen(['docker', 'push', target_image])
        rc = p.wait()
        if rc != 0:
            raise RuntimeError('Failed to push release image')
    finally:
        shutil.rmtree(tempdir)
        # cleanup the temporary image
        p = subprocess.Popen(['docker', 'rmi', release_image])
        rc = p.wait()
        if rc != 0:
            print('warning: failed to delete temporary image')
        

def _build_dockerfile(image, config):
    dockerfile = ["FROM "+image]
    for k, v in config.items():
        dockerfile.append("ENV {} {}".format(k.upper(), v))
    return '\n'.join(dockerfile)
