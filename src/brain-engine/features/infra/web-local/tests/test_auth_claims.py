import json
from pathlib import Path
import subprocess
import unittest

class AuthClaimsContractTest(unittest.TestCase):
    def test_google_develop_emits_the_audience_and_role_required_by_api(self):
        overlay = Path(__file__).resolve().parents[2] / 'auth-google-develop' / 'compose.auth-google-develop.yaml'
        config = json.loads(subprocess.check_output(['ruby', '-ryaml', '-rjson', '-e', 'puts JSON.generate(YAML.load_file(ARGV[0]))', str(overlay)], text=True))
        patch = Path(__file__).resolve().parents[1] / 'operations' / 'compose.auth-claims.yaml'
        patch_config = json.loads(subprocess.check_output(['ruby', '-ryaml', '-rjson', '-e', 'puts JSON.generate(YAML.load_file(ARGV[0]))', str(patch)], text=True))
        self.assertEqual(patch_config['services']['auth-develop']['environment'], {
            'GOTRUE_JWT_AUD': 'authenticated',
            'GOTRUE_JWT_DEFAULT_GROUP_NAME': 'authenticated',
        })
        self.assertEqual(set(patch_config['services']), {'auth-develop'})
        env = config['services']['auth-develop']['environment']
        self.assertEqual(env.get('GOTRUE_JWT_AUD'), 'authenticated')
        self.assertEqual(env.get('GOTRUE_JWT_DEFAULT_GROUP_NAME'), 'authenticated')
        self.assertEqual(set(config['services']), {'auth-develop'})
