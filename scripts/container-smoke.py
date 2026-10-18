import argparse, json, re, subprocess, time, urllib.request, urllib.error, uuid
parser = argparse.ArgumentParser()
parser.add_argument('--image', default='format:dhi-verified')
parser.add_argument('--port', type=int, default=19087)
options = parser.parse_args()
name = 'format-smoke-' + uuid.uuid4().hex[:8]
args = ['docker', 'run', '--rm', '-d', '--name', name, '-p', f'127.0.0.1:{options.port}:8080']
for key, value in {
    'SESSION_SECRET': 'local-smoke-secret-at-least-thirty-two-characters',
    'APP_BASE_URL': f'http://localhost:{options.port}',
    'GOOGLE_OAUTH_CLIENT_ID': 'local-test',
    'GOOGLE_OAUTH_CLIENT_SECRET': 'local-test',
    'R2_ACCOUNT_ID': 'local-test',
    'R2_ACCESS_KEY_ID': 'local-test',
    'R2_SECRET_ACCESS_KEY': 'local-test',
    'R2_BUCKET': 'local-test',
    'R2_PUBLIC_BASE_URL': 'https://cdn.example.test',
}.items():
    args += ['-e', f'{key}={value}']
args += [options.image]
subprocess.run(args, check=True, capture_output=True)
base = f'http://127.0.0.1:{options.port}'
try:
    deadline = time.monotonic() + 35
    while True:
        try:
            with urllib.request.urlopen(base + '/healthz', timeout=2) as r:
                assert json.load(r)['status'] == 'ok'
                break
        except (OSError, ValueError):
            if time.monotonic() > deadline:
                raise RuntimeError(subprocess.check_output(['docker', 'logs', name], stderr=subprocess.STDOUT).decode())
            time.sleep(.25)
    print('PASS container startup and /healthz')
    subprocess.run(['docker', 'exec', name, 'wget', '--no-verbose', '--tries=1', '--spider', 'http://localhost:8080/healthz'], check=True, capture_output=True)
    print('PASS configured container health check')
    with urllib.request.urlopen(base + '/') as r:
        html = r.read().decode()
        assert r.status == 200 and 'format.hackclub.com' in html
    print('PASS built HTML page')
    assets = re.findall(r'(?:src|href)="(/_next/[^\"]+\.(?:js|css))"', html)
    assert assets, 'No built assets found'
    for path in set(assets):
        with urllib.request.urlopen(base + path) as r:
            assert r.status == 200 and len(r.read()) > 0
    print(f'PASS {len(set(assets))} built JS/CSS assets')
    for path, method in [('/api/auth/me', 'GET'), ('/api/assets', 'POST'), ('/api/html/transform', 'POST')]:
        try:
            urllib.request.urlopen(urllib.request.Request(base + path, method=method))
            raise AssertionError(f'{path} allowed an unauthenticated request')
        except urllib.error.HTTPError as e:
            assert e.code == 401, (path, e.code)
    print('PASS protected API endpoints reject unauthenticated requests')
    with urllib.request.urlopen(base + '/api/config') as r:
        assert json.load(r)['cdnBaseUrl'] == 'https://cdn.example.test'
    print('PASS public configuration')
    uid = subprocess.check_output(['docker', 'exec', name, 'id', '-u'], text=True).strip()
    assert uid == '65532', uid
    print('PASS DHI non-root runtime user (65532)')
    os_release = subprocess.check_output(['docker', 'exec', name, 'cat', '/etc/os-release'], text=True)
    assert 'Docker Hardened Images' in os_release, os_release
    apk = subprocess.run(['docker', 'exec', name, 'apk', '--version'], capture_output=True)
    assert apk.returncode != 0, 'Runtime unexpectedly contains the apk package manager'
    print('PASS DHI runtime base without package manager')
    vips_version = subprocess.check_output(['docker', 'exec', name, 'vips', '--version'], text=True).strip()
    oxipng_version = subprocess.check_output(['docker', 'exec', name, 'oxipng', '--version'], text=True).strip()
    assert vips_version == 'vips-8.18.7', vips_version
    assert oxipng_version == 'oxipng 10.2.1', oxipng_version
    print(f'PASS native image libraries: {vips_version}, {oxipng_version}')
    for command in [
        ['vips', 'black', '/tmp/format-smoke.png', '32', '32', '--bands', '3'],
        ['vips', 'copy', '/tmp/format-smoke.png', '/tmp/format-smoke.jpg'],
        ['oxipng', '-o', '1', '--strip', 'safe', '-i', 'off', '/tmp/format-smoke.png'],
    ]:
        subprocess.run(['docker', 'exec', name, *command], check=True, capture_output=True)
    width = subprocess.check_output(['docker', 'exec', name, 'vipsheader', '-f', 'width', '/tmp/format-smoke.jpg'], text=True).strip()
    assert width == '32', width
    print('PASS runtime PNG/JPEG encoding and PNG optimization')
    health_deadline = time.monotonic() + 40
    while True:
        status = subprocess.check_output(['docker', 'inspect', '--format', '{{.State.Health.Status}}', name], text=True).strip()
        if status == 'healthy':
            break
        if status == 'unhealthy' or time.monotonic() >= health_deadline:
            raise RuntimeError(f'Container health status: {status}')
        time.sleep(1)
    print('PASS Docker health status is healthy')
finally:
    subprocess.run(['docker', 'stop', name], check=False, capture_output=True)
