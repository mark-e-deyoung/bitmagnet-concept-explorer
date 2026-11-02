import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

MODULE = Path(__file__).with_name('build_image.py')
spec = importlib.util.spec_from_file_location('artifact_builder', MODULE)
builder = importlib.util.module_from_spec(spec)
spec.loader.exec_module(builder)


class ArtifactIdentityTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'VERSION').write_text('0.0.3-rc.1\n')
        (self.root / 'Containerfile').write_text('ARG GO_BUILDER_IMAGE=golang:1.27.1-bookworm@sha256:' + 'a' * 64 + '\n')
        self.git('init', '-q')
        self.commit()

    def git(self, *args):
        return subprocess.check_output(['git', *args], cwd=self.root, text=True, stderr=subprocess.DEVNULL).strip()

    def commit(self):
        self.git('add', '.')
        self.git('-c', 'user.name=Test', '-c', 'user.email=test@example.invalid', 'commit', '-qm', 'synthetic fixture')

    def test_identity_uses_full_source_and_pinned_builder(self):
        result = builder.metadata(self.root)
        self.assertEqual(result['source_revision'], self.git('rev-parse', 'HEAD'))
        self.assertEqual(result['version'], '0.0.3-rc.1')
        self.assertTrue(result['builder'].endswith('@sha256:' + 'a' * 64))

    def test_dirty_source_and_floating_builder_are_rejected(self):
        (self.root / 'VERSION').write_text('0.0.4\n')
        with self.assertRaisesRegex(ValueError, 'clean checkout'):
            builder.metadata(self.root)
        (self.root / 'Containerfile').write_text('ARG GO_BUILDER_IMAGE=golang:1.27.1-bookworm\n')
        self.commit()
        with self.assertRaisesRegex(ValueError, 'digest-pinned'):
            builder.metadata(self.root)

    def test_invalid_version_is_rejected(self):
        (self.root / 'VERSION').write_text('release candidate\n')
        self.commit()
        with self.assertRaisesRegex(ValueError, 'VERSION'):
            builder.metadata(self.root)


if __name__ == '__main__':
    unittest.main()
