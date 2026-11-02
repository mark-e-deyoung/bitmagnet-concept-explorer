"""Build a local qualification image with verified source/version/builder identity.

No push, registry publication or deployment. A failed pinned build is terminal.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[2]


def metadata(root):
    env = {k: v for k, v in os.environ.items() if not k.startswith('GIT_')}
    def git(*args):
        return subprocess.check_output(['git', *args], cwd=root, env=env, text=True).strip()
    revision = git('rev-parse', '--verify', 'HEAD^{commit}')
    if len(revision) not in (40, 64) or not re.fullmatch('[0-9a-f]+', revision):
        raise ValueError('source revision must be a full Git commit identity')
    if git('status', '--porcelain', '--untracked-files=normal'):
        raise ValueError('qualification build requires a clean checkout')
    version = (root / 'VERSION').read_text(encoding='utf-8').strip()
    if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?', version):
        raise ValueError('VERSION must be a safe release/candidate version')
    containerfile = (root / 'Containerfile').read_text(encoding='utf-8')
    match = re.search(r'^ARG GO_BUILDER_IMAGE=(\S+)$', containerfile, re.M)
    if not match or not re.fullmatch(r'golang:[A-Za-z0-9.-]+@sha256:[0-9a-f]{64}', match[1]):
        raise ValueError('builder must be the reviewed digest-pinned official Go image')
    return dict(version=version, source_revision=revision, builder=match[1],
                source='https://github.com/mark-e-deyoung/bitmagnet-concept-explorer')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--metadata-only', action='store_true')
    parser.add_argument('--image')
    parser.add_argument('--evidence', type=Path)
    args = parser.parse_args()
    identity = metadata(ROOT)
    identity['image'] = args.image or 'ghcr.io/mark-e-deyoung/bitmagnet-concept-explorer:' + identity['version']
    if args.metadata_only:
        identity['mode'] = 'source_metadata_only_not_a_build_receipt'
    else:
        subprocess.run(['docker', 'build', '--build-arg', 'GO_BUILDER_IMAGE=' + identity['builder'],
                        '--build-arg', 'RELEASE_VERSION=' + identity['version'],
                        '--build-arg', 'SOURCE_REVISION=' + identity['source_revision'],
                        '-f', str(ROOT / 'Containerfile'), '-t', identity['image'], str(ROOT)], check=True)
        image = json.loads(subprocess.check_output(['docker', 'image', 'inspect', identity['image']]))[0]
        labels = image['Config'].get('Labels', {})
        for key, value in [('version', identity['version']), ('revision', identity['source_revision']), ('source', identity['source'])]:
            if labels.get('org.opencontainers.image.' + key) != value:
                raise ValueError('built OCI ' + key + ' does not match selected source')
        identity.update(mode='local_image_build_and_identity_verified', image_id=image['Id'],
                        platform=image['Os'] + '/' + image['Architecture'], labels=labels,
                        repo_digests=image.get('RepoDigests', []))
        lock = ROOT / 'package-lock.json'
        if lock.exists():
            identity['npm_lock_sha256'] = hashlib.sha256(lock.read_bytes()).hexdigest()
    if args.evidence:
        args.evidence.parent.mkdir(parents=True, exist_ok=True)
        args.evidence.write_text(json.dumps(identity, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(identity, indent=2))


if __name__ == '__main__':
    main()
