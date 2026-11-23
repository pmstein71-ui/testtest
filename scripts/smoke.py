"""Real-process smoke tests; run after building dist/pulse-{server,client}[.exe]."""
from contextlib import closing
import sys
import json
import os
from pathlib import Path
import re
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.request
import urllib.error
import traceback

ROOT = Path(__file__).resolve().parents[1]
EXT = '.exe' if os.name == 'nt' else ''

def request(base, path, body=None, token='', headers=None):
    h = {'Content-Type': 'application/json', 'X-Pulse-Client': '1', 'Authorization': 'Bearer ' + token}
    h.update(headers or {})
    req = urllib.request.Request(base + path, data=json.dumps(body).encode() if body is not None else None, headers=h)
    try:
        with urllib.request.urlopen(req, timeout=10) as r:
            return r.status, r.read()
    except urllib.error.HTTPError as e:
        return e.code, e.read()

def wait_for(check, processes):
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        for p in processes:
            assert p.poll() is None, f'Process exited unexpectedly: {p.returncode}'
        try:
            result = check()
            if result:
                return result
        except (OSError, ValueError):
            pass
        time.sleep(.1)
    raise AssertionError('Timed out waiting for readiness')

def main():
    with tempfile.TemporaryDirectory() as d:
        work = Path(d)
        env = dict(os.environ, DATA_DIR=str(work / 'data'), ADMIN_PASSWORD='smoke-test-password-123',
                   ADMIN_LOGIN='admin', PULSE_NO_BROWSER='1', PULSE_NO_DIALOG='1', BACKUP_INTERVAL='1m', TLS_CERT='', TLS_KEY='')
        # Ask OS for a free port. Process readiness checks detect bind errors.
        with socket.socket() as s:
            s.bind(('127.0.0.1', 0))
            port = s.getsockname()[1]
        base = f'http://127.0.0.1:{port}'
        env.update(LISTEN_ADDR=f'127.0.0.1:{port}', SERVER_URL=base)
        def launch(name, overrides=None):
            return subprocess.Popen([str(ROOT / 'dist' / ('pulse-' + name + EXT))], cwd=work,
                                    env=dict(env, **(overrides or {})), stdin=subprocess.DEVNULL,
                                    stdout=log, stderr=log)
        with (work / 'process.log').open('w+', encoding='utf-8') as log:
            processes = []
            try:
                # Failure paths must be nonzero, with an actionable diagnostic.
                for name, overrides in [('server', {'ADMIN_PASSWORD': ''}),
                                        ('server', {'TLS_CERT': 'missing.pem'}),
                                        ('client', {'SERVER_URL': 'http://remote.invalid:8080'})]:
                    p = launch(name, overrides)
                    assert p.wait(timeout=20) != 0, f'{name}: invalid config accepted'
                server = launch('server'); processes.append(server)
                wait_for(lambda: request(base, '/api/me')[0] == 401, processes)
                snapshots = wait_for(lambda: list((work / 'data' / 'backups').glob('*.db')), processes)
                with closing(sqlite3.connect(snapshots[0])) as db:
                    assert db.execute('PRAGMA integrity_check').fetchone()[0] == 'ok'
                    assert db.execute('SELECT login FROM users').fetchone()[0] == 'admin'
                client = launch('client'); processes.append(client)
                def client_url():
                    text = (work / 'process.log').read_text(encoding='utf-8', errors='replace')
                    matches = re.findall(r'Open (http://127\.0\.0\.1:\d+)', text)
                    return matches[-1] if matches else None
                local = wait_for(client_url, processes)
                assert request(local, '/')[0] == 200
                assert request(local, '/api/me', headers={'Origin': 'https://evil.invalid'})[0] == 403
                assert request(local, '/api/me', headers={'X-Pulse-Client': ''})[0] == 403
                code, data = request(local, '/api/login', {'login': 'admin', 'password': env['ADMIN_PASSWORD']})
                assert code == 200, (code, data)
                token = json.loads(data)['token']
                assert request(local, '/api/me', token=token)[0] == 200
                assert request(local, '/api/logout', {}, token)[0] == 204
                assert request(local, '/api/me', token=token)[0] == 401
                server.terminate(); server.wait(timeout=15); processes.remove(server)
                assert request(local, '/api/me')[0] == 502
                # Persisted database starts without bootstrap password.
                server = launch('server', {'ADMIN_PASSWORD': ''}); processes.append(server)
                wait_for(lambda: request(base, '/api/me')[0] == 401, processes)
                # Wrong credentials and subsequent valid login through proxy.
                assert request(local, '/api/login', {'login': 'admin', 'password': 'wrong'})[0] == 401
                time.sleep(1.1)
                assert request(local, '/api/login', {'login': 'admin', 'password': env['ADMIN_PASSWORD']})[0] == 200
                print('PASS: startup errors, DB restart, snapshot integrity, client/proxy, login/me/logout, server offline')
            except BaseException:
                report = (work / 'process.log').read_text(encoding='utf-8', errors='replace') + '\n' + traceback.format_exc()
                print(report)
                print('::error::' + report.replace('%', '%25').replace('\r', '%0D').replace('\n', '%0A'))
                raise
            finally:
                for p in processes:
                    if p.poll() is None:
                        p.terminate(); p.wait(timeout=15)


if __name__ == '__main__':
    sys.stdout.reconfigure(encoding='utf-8')
    try:
        main()
    except BaseException:
        report = traceback.format_exc()
        print('::error::' + report.replace('%', '%25').replace('\r', '%0D').replace('\n', '%0A'))
        raise
