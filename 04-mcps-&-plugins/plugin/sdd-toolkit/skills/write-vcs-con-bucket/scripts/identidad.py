#!/usr/bin/env python3
"""Muestra solo el client_email de la credencial del server, nunca la key ni su ruta."""
import json
import os
from pathlib import Path
import sys


def main():
    credential = os.environ.get("GOOGLE_APPLICATION_CREDENTIALS")
    if not credential:
        sys.exit("FALTA: GOOGLE_APPLICATION_CREDENTIALS; no se confirmó la cuenta lectora.")
    try:
        account = json.loads(Path(credential).read_text()).get("client_email")
    except (OSError, ValueError, AttributeError):
        sys.exit("FALTA: no se pudo leer la identidad de la credencial del server.")
    if not isinstance(account, str) or not account.endswith(".iam.gserviceaccount.com"):
        sys.exit("FALTA: la credencial no identifica una service account.")
    print(account)


if __name__ == "__main__":
    main()
