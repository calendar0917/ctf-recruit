## Edge (VPS) Deployment Skeleton

This folder documents a pragmatic deployment model for:

- Internal PVE VM/LXC running the full platform (Compose in `deploy/docker-compose.prod.yml`)
- A public VPS acting as the edge:
  - TLS termination for the main site (`ctf.yulinsec.cn`) via Caddy
  - `frps` server for tunneling to the internal node
  - TCP port-range forwarding for dynamic instances (`20000-20499`)

This keeps the internal node off the public Internet while still allowing public access.

### DNS

Create A records pointing to the public VPS:

- `ctf.yulinsec.cn`
- `inst.yulinsec.cn`

### Ports to Open On the Public VPS

- `80/tcp` and `443/tcp` (Caddy)
- `7000/tcp` (frps bind port, configurable)
- `20000-20499/tcp` (dynamic instance ports, configurable)

### Internal (PVE) Node Setup

On the internal VM/LXC, run the platform with the following runtime variables:

- `PUBLIC_BASE_URL=https://ctf.yulinsec.cn`
- `RUNTIME_PUBLIC_BASE_URL=http://inst.yulinsec.cn`
- `RUNTIME_PORT_MIN=20000`
- `RUNTIME_PORT_MAX=20499`
- `RUNTIME_BIND_ADDR=127.0.0.1`

Then start the production compose:

```bash
export POSTGRES_PASSWORD='...'
export JWT_SECRET='...'
export PUBLIC_BASE_URL='https://ctf.yulinsec.cn'
export RUNTIME_PUBLIC_BASE_URL='http://inst.yulinsec.cn'
export RUNTIME_PORT_MIN='20000'
export RUNTIME_PORT_MAX='20499'
export RUNTIME_BIND_ADDR='127.0.0.1'
make prod-compose-up
```

Make sure the internal node runs `frpc` (see `frpc.ini` template in this folder).

Important:

- `frpc` needs one TCP proxy per port. For `20000-20499` that is 500 entries.
- In practice, generate the config on your own machine and keep it next to your deployment.

Example generator:

```bash
for p in $(seq 20000 20499); do
  cat <<EOF
[ctf_instance_${p}]
type = tcp
local_ip = 127.0.0.1
local_port = ${p}
remote_port = ${p}

EOF
done
```

### Public VPS Setup

1. Install Caddy and frps.
2. Use the templates in this folder:
   - `Caddyfile` for `ctf.yulinsec.cn` TLS termination and reverse proxy
   - `frps.ini` for frp server
3. Start `frps`, then start Caddy.

Notes:

- Dynamic instances are exposed as `http://inst.yulinsec.cn:<port>`.
- Keep the main site on HTTPS. Avoid HSTS on `inst.yulinsec.cn` if you keep instances on plain HTTP.
