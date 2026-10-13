#!/usr/bin/env python3
"""Lee únicamente metadata nativa con la misma credencial lectora que el MCP."""
import argparse
from datetime import datetime, timezone
import json
import os
import subprocess
import sys


def read_metadata(bucket, object_name):
    credential = os.environ.get("GOOGLE_APPLICATION_CREDENTIALS")
    if not credential:
        raise ValueError("FALTA: GOOGLE_APPLICATION_CREDENTIALS; no se usa la cuenta activa de gcloud.")
    if not bucket or "/" in bucket or not object_name or "#" in object_name:
        raise ValueError("FALTA: bucket sin gs:// y nombre de objeto sin selector de generación.")
    environment = os.environ.copy()
    environment["CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE"] = credential
    try:
        result = subprocess.run(
            ["gcloud", "storage", "objects", "describe", f"gs://{bucket}/{object_name}",
             "--raw", "--format=json", "--quiet"],
            env=environment, capture_output=True, text=True, timeout=45, check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        raise ValueError("FALTA: gcloud no está disponible o la consulta de metadata venció.") from None
    if result.returncode:
        # El stderr de gcloud puede contener la ruta de la key: no se transmite.
        raise ValueError("FALTA: la cuenta lectora no pudo consultar la metadata nativa.")
    try:
        raw = json.loads(result.stdout)
        metadata = {key: raw[key] for key in ("bucket", "name", "generation", "updated")}
        for key in ("md5Hash", "crc32c"):
            if key in raw:
                metadata[key] = raw[key]
        if (metadata["bucket"] != bucket or metadata["name"] != object_name
                or not isinstance(metadata["generation"], str)
                or not metadata["generation"].isdigit()
                or not isinstance(metadata["updated"], str) or not metadata["updated"]):
            raise ValueError
        # gcloud formatea RFC 3339 como +00:00 con seis decimales; el MCP usa Z
        # con milisegundos. Normalizar sin perder precisión permite comparar ambos.
        timestamp = datetime.fromisoformat(metadata["updated"])
        if timestamp.tzinfo is None or timestamp.microsecond % 1000:
            raise ValueError
        metadata["updated"] = timestamp.astimezone(timezone.utc).isoformat(
            timespec="milliseconds"
        ).replace("+00:00", "Z")
    except (ValueError, KeyError, TypeError):
        raise ValueError("FALTA: respuesta de metadata incompleta o de otro objeto.") from None
    return metadata


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("bucket")
    parser.add_argument("object_name")
    args = parser.parse_args()
    try:
        metadata = read_metadata(args.bucket, args.object_name)
    except ValueError as error:
        sys.exit(str(error))
    print(json.dumps(metadata, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
