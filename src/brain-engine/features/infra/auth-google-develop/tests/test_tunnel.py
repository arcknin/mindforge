from __future__ import annotations

import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "open-develop-tunnels.py"
SSH_VALUES = {
    "VM_SSH_HOST": "vm.mindforge.test",
    "VM_SSH_PORT": "2222",
    "VM_SSH_USER": "mindforge-dev",
}


def write_private_file(path: Path, values: dict[str, str]) -> Path:
    path.write_text("".join(f"{key}={value}\n" for key, value in values.items()), encoding="utf-8")
    path.chmod(0o600)
    return path


class TunnelHelperTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="mindforge-tunnel-")
        self.directory = Path(self.temp.name)
        self.bin_dir = self.directory / "bin"
        self.bin_dir.mkdir()
        self.record = self.directory / "ssh-args.json"
        self.fake_ssh = self.bin_dir / "ssh"
        self.fake_ssh.write_text(
            "#!/usr/bin/env python3\n"
            "import json, os, sys\n"
            "open(os.environ['FAKE_SSH_RECORD'], 'w', encoding='utf-8').write(json.dumps(sys.argv[1:]))\n"
            "raise SystemExit(int(os.environ.get('FAKE_SSH_STATUS', '0')))\n",
            encoding="utf-8",
        )
        self.fake_ssh.chmod(0o755)
        self.env_file = write_private_file(self.directory / "ssh.env.private", SSH_VALUES)
        self.environment = os.environ.copy()
        self.environment["PATH"] = f"{self.bin_dir}:{self.environment.get('PATH', '')}"
        self.environment["FAKE_SSH_RECORD"] = str(self.record)

    def tearDown(self):
        self.temp.cleanup()

    def run_tunnel(self, env: dict[str, str] | None = None):
        return subprocess.run(
            ["python3", str(SCRIPT), "--file", str(self.env_file)],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=env or self.environment,
            check=False,
            timeout=10,
        )

    def test_tunnel_uses_only_the_approved_api_loopback_forward_and_strict_host_verification(self):
        result = self.run_tunnel()
        self.assertEqual(result.returncode, 0, result.stderr)
        args = json.loads(self.record.read_text(encoding="utf-8"))
        self.assertIn("-F", args)
        self.assertEqual(args[args.index("-F") + 1], "/dev/null")
        self.assertIn("-N", args)
        self.assertIn("-T", args)
        self.assertIn("-o", args)
        self.assertIn("StrictHostKeyChecking=yes", args)
        self.assertIn("ExitOnForwardFailure=yes", args)
        self.assertIn("-p", args)
        self.assertEqual(args[args.index("-p") + 1], "2222")
        self.assertIn("127.0.0.1:11001:127.0.0.1:18087", args)
        self.assertNotIn("127.0.0.1:11000:127.0.0.1:10000", args)
        self.assertEqual(args[-1], "mindforge-dev@vm.mindforge.test")
        self.assertFalse(any(arg in args for arg in ("-R", "sshpass", "password")))
        self.assertEqual(len(args), 13)

    def test_forward_failure_is_returned_without_attempting_to_kill_existing_processes(self):
        env = self.environment.copy()
        env["FAKE_SSH_STATUS"] = "255"
        result = self.run_tunnel(env)
        self.assertEqual(result.returncode, 255)
        args = json.loads(self.record.read_text(encoding="utf-8"))
        self.assertIn("ExitOnForwardFailure=yes", args)
        self.assertNotIn("kill", args)
        self.assertNotIn("pkill", args)

    def test_invalid_private_file_fails_before_ssh_is_invoked(self):
        values = dict(SSH_VALUES)
        values["VM_SSH_HOST"] = "-oProxyCommand=touch /tmp/unsafe"
        write_private_file(self.env_file, values)
        result = self.run_tunnel()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.record.exists())
        self.assertNotIn("-oProxyCommand", result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
