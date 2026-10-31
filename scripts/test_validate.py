"""Check module discovery and failure propagation without needing a Go SDK."""

import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from validate import validate_go


class GoValidation(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        for name in ("go.mod", "nested/go.mod", "main.go"):
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.touch()

    @patch("validate.subprocess.run")
    @patch("validate.subprocess.check_output")
    def test_all_modules(self, output, run):
        output.side_effect = [b"go.mod\0nested/go.mod\0main.go\0", ""]
        validate_go(self.root)
        self.assertEqual(run.call_count, 6)
        self.assertEqual({c.kwargs["cwd"] for c in run.call_args_list},
                         {self.root, self.root / "nested"})
        self.assertTrue(all(c.kwargs["check"] for c in run.call_args_list))

    @patch("validate.subprocess.check_output")
    def test_rejects_unformatted_code(self, output):
        output.side_effect = [b"main.go\0", "main.go\n"]
        with self.assertRaisesRegex(ValueError, "Unformatted"):
            validate_go(self.root)

    @patch("validate.subprocess.run", side_effect=subprocess.CalledProcessError(1, "go"))
    @patch("validate.subprocess.check_output", return_value=b"go.mod\0")
    def test_go_failure_propagates(self, output, run):
        with self.assertRaises(subprocess.CalledProcessError):
            validate_go(self.root)
