#!/usr/bin/env python3
"""Pruebas locales del plugin: no llama a GCS ni necesita credenciales."""
import json
import importlib.util
import os
from pathlib import Path
import re
import subprocess
import tempfile
from types import SimpleNamespace
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[1]
PLUGIN = ROOT / "plugin/sdd-toolkit"
PREFIX = "mcp__plugin_sdd-toolkit_gcs__"
BUCKET = "bucket-sintetico-prueba"


def check(label, script, event, expected, env):
    result = subprocess.run(
        [str(script)], input=json.dumps(event), text=True,
        capture_output=True, env=env, check=False,
    )
    assert result.returncode == expected, (
        f"{label}: esperado {expected}, recibido {result.returncode}: {result.stderr}"
    )
    print(f"PASS · {label} · exit {result.returncode}")


def main():
    env = os.environ.copy()
    env.pop("GCSGREP_BUCKET", None)
    env["CLAUDE_PLUGIN_ROOT"] = str(PLUGIN)
    env["CLAUDE_PLUGIN_OPTION_BUCKET"] = BUCKET
    hooks = json.loads((PLUGIN / "hooks/hooks.json").read_text())["hooks"]["PreToolUse"]
    matcher = next(x["matcher"] for x in hooks if "mcp__" in x["matcher"])
    for tool in ("list_objects", "read_object_metadata", "read_object_content",
                 "write_object_safe", "create_bucket", "delete_object", "tool_futura"):
        assert re.fullmatch(matcher, PREFIX + tool), f"El matcher no alcanza {tool}"
        check(tool, PLUGIN / "hooks/gcs-allowlist.sh", {
            "tool_name": PREFIX + tool, "tool_input": {"bucket_name": BUCKET},
        }, 0 if tool in ("list_objects", "read_object_metadata", "read_object_content") else 2, env)
    check("otro bucket", PLUGIN / "hooks/gcs-allowlist.sh", {
        "tool_name": PREFIX + "list_objects", "tool_input": {"bucket_name": "otro"},
    }, 2, env)
    check("bucket ausente", PLUGIN / "hooks/gcs-allowlist.sh", {
        "tool_name": PREFIX + "list_objects", "tool_input": {},
    }, 2, env)
    missing = env.copy()
    missing.pop("CLAUDE_PLUGIN_OPTION_BUCKET")
    check("sin configuración", PLUGIN / "hooks/gcs-allowlist.sh", {
        "tool_name": PREFIX + "list_objects", "tool_input": {"bucket_name": BUCKET},
    }, 2, missing)
    conflicting = env.copy()
    conflicting["GCSGREP_BUCKET"] = "otro"
    check("userConfig prevalece sobre el shell", PLUGIN / "hooks/gcs-allowlist.sh", {
        "tool_name": PREFIX + "list_objects", "tool_input": {"bucket_name": BUCKET},
    }, 0, conflicting)

    # Otro repo, con espacios en la ruta y sin .claude/: prueba la portabilidad del gate.
    with tempfile.TemporaryDirectory(prefix="tp4 proyecto ") as directory:
        project = Path(directory)
        subprocess.run(["git", "init", "-q", str(project)], check=True)
        env["CLAUDE_PROJECT_DIR"] = str(project)
        valid = "# Demo\n#### FR-1 · Mostrar un resultado\n> **VC-1** — comando sale 0.\n"
        invalid = "# Demo\n#### FR-1 · Mostrar un resultado\n"
        spec = project / "demo-spec.md"
        agent = project / "plugin/sdd-toolkit/agents/review-spec.md"
        agent.parent.mkdir(parents=True)
        agent.write_text((PLUGIN / "agents/review-spec.md").read_text())
        gate = PLUGIN / "hooks/spec-vc-gate.sh"
        for label, content, expected in (("Write con cobertura", valid, 0),
                                         ("Write sin VC", invalid, 2)):
            check(label, gate, {"tool_name": "Write", "tool_input": {
                "file_path": str(spec), "content": content,
            }}, expected, env)
        check("definición del agente fuera del gate de specs", gate, {
            "tool_name": "Write", "tool_input": {
                "file_path": str(agent), "content": agent.read_text(),
            },
        }, 0, env)
        spec.write_text(invalid)
        check("commit con spec nueva sin VC", gate, {
            "tool_name": "Bash", "tool_input": {"command": "git add . && git commit -m demo"},
        }, 2, env)
        spec.write_text(valid)
        subprocess.run(["git", "-C", str(project), "add", "demo-spec.md"], check=True)
        check("commit con cobertura", gate, {
            "tool_name": "Bash", "tool_input": {"command": "git commit -m demo"},
        }, 0, env)
        spec.write_text(invalid)
        subprocess.run(["git", "-C", str(project), "add", "demo-spec.md"], check=True)
        spec.write_text(valid)
        check("staging sin VC aunque disco tenga cobertura", gate, {
            "tool_name": "Bash", "tool_input": {"command": "git commit -m demo"},
        }, 2, env)
    # El helper no puede usar la cuenta activa ni transmitir metadata libre o stderr.
    module_spec = importlib.util.spec_from_file_location(
        "procedencia", PLUGIN / "skills/write-vcs-con-bucket/scripts/procedencia.py"
    )
    helper = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(helper)
    raw = {"bucket": BUCKET, "name": "logs/app/api.log", "generation": "123",
           "updated": "2026-10-13T18:02:46.750000+00:00", "md5Hash": "fixture-md5",
           "metadata": {"instruccion": "TEXTO_HOSTIL_NO_TRANSMITIR"}}
    credential = "/credencial-sintetica.json"
    with patch.dict(os.environ, {"GOOGLE_APPLICATION_CREDENTIALS": credential,
                                 "CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE": "cuenta-activa"}):
        with patch.object(helper.subprocess, "run", return_value=SimpleNamespace(
                returncode=0, stdout=json.dumps(raw))) as run:
            result = helper.read_metadata(BUCKET, "logs/app/api.log")
            assert result["updated"] == "2026-10-13T18:02:46.750Z"
            assert "metadata" not in result
            assert run.call_args.args[0][1:4] == ["storage", "objects", "describe"]
            assert "--raw" in run.call_args.args[0]
            assert run.call_args.kwargs["env"]["CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE"] == credential
            assert os.environ["CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE"] == "cuenta-activa"
        print("PASS · metadata nativa: misma credencial, fecha normalizada y salida acotada")
        with patch.object(helper.subprocess, "run", return_value=SimpleNamespace(
                returncode=1, stderr="TOKEN_INTERNO Y RUTA_DE_KEY")):
            try:
                helper.read_metadata(BUCKET, "logs/app/api.log")
                raise AssertionError("Se aceptó una consulta fallida")
            except ValueError as error:
                assert "TOKEN_INTERNO" not in str(error) and "RUTA_DE_KEY" not in str(error)
        print("PASS · error de gcloud sin transmitir token ni ruta de key")
        for label, replacement in (("objeto ajeno", {"name": "otro.log"}),
                                   ("precisión no soportada", {"updated": "2026-10-13T18:02:46.750001Z"})):
            with patch.object(helper.subprocess, "run", return_value=SimpleNamespace(
                    returncode=0, stdout=json.dumps(raw | replacement))):
                try:
                    helper.read_metadata(BUCKET, "logs/app/api.log")
                    raise AssertionError(label)
                except ValueError:
                    pass
            print(f"PASS · metadata rechazada: {label}")
    with patch.dict(os.environ, {}, clear=True), patch.object(helper.subprocess, "run") as run:
        try:
            helper.read_metadata(BUCKET, "logs/app/api.log")
            raise AssertionError("Se usó autenticación implícita")
        except ValueError:
            run.assert_not_called()
    print("PASS · sin credencial no hay consulta ni fallback a la cuenta activa")
    print("RESULTADO: 22 casos PASS; sin llamadas al bucket ni commits reales.")


if __name__ == "__main__":
    main()
