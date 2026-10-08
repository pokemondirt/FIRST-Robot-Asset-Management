FIRST Robotics Inventory - {{OS}}/{{ARCH}}
==================================================

Run:
  chmod +x first-inventory
  ./first-inventory

Then open:
  http://127.0.0.1:8000/#/checkout

Data
  The database lives in ./data/inventory.db next to the binary. Keep the data
  folder when upgrading. Backups go to ./data/backups/.

LAN
  The server listens on all interfaces, so other machines on the same network
  can open http://<this-host-ip>:8000/#/checkout - nothing to install there.
  The startup log prints the exact address.

  There is no login. Anyone who can reach that address can change the stock,
  so keep it on a trusted warehouse network and never expose it to the
  internet.
