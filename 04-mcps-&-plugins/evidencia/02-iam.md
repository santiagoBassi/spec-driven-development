# 02 · IAM del bucket

Capturado el 2026-10-12T21:38Z con la cuenta dueña del proyecto (`gcloud` CLI 588.0.0).

- Bucket: `gs://sdd-fardenghi-itba`
- Proyecto: `sat-spec-driven-development`
- Cuenta del server MCP: `lectora@sat-spec-driven-development.iam.gserviceaccount.com`
- Cuenta para el caso "objeto sin permisos": `sin-acceso@sat-spec-driven-development.iam.gserviceaccount.com`

**Resultado:** `lectora` tiene **solo** `roles/storage.objectViewer`, y **solo sobre el
bucket**. No tiene ningún rol a nivel proyecto. `sin-acceso` no aparece en ninguna de
las dos políticas.

## Política IAM del bucket

```console
$ gcloud storage buckets get-iam-policy gs://sdd-fardenghi-itba
bindings:
- members:
  - projectEditor:sat-spec-driven-development
  - projectOwner:sat-spec-driven-development
  role: roles/storage.legacyBucketOwner
- members:
  - projectViewer:sat-spec-driven-development
  role: roles/storage.legacyBucketReader
- members:
  - projectEditor:sat-spec-driven-development
  - projectOwner:sat-spec-driven-development
  role: roles/storage.legacyObjectOwner
- members:
  - projectViewer:sat-spec-driven-development
  role: roles/storage.legacyObjectReader
- members:
  - serviceAccount:lectora@sat-spec-driven-development.iam.gserviceaccount.com   # <-- la cuenta del server
  role: roles/storage.objectViewer                                                # <-- solo lectura de objetos
etag: CAI=
```

Los cuatro bindings `legacy*` son los que GCS crea por defecto para los roles básicos
del proyecto (owner/editor/viewer). `lectora` no es owner, editor ni viewer del proyecto
(ver abajo), así que no hereda ninguno de ellos.

## Roles de `lectora` a nivel proyecto

```console
$ gcloud projects get-iam-policy sat-spec-driven-development \
    --flatten="bindings[].members" \
    --filter="bindings.members:lectora@sat-spec-driven-development.iam.gserviceaccount.com" \
    --format="table(bindings.role)"
$
```

Salida vacía: ningún rol.

Para mostrar que el filtro no está vacío por error, la política completa del proyecto:

```console
$ gcloud projects get-iam-policy sat-spec-driven-development \
    --flatten="bindings[].members" --format="table(bindings.role,bindings.members)"
ROLE                                            MEMBERS
roles/compute.instanceGroupManagerServiceAgent  serviceAccount:430748831547@cloudservices.gserviceaccount.com
roles/compute.serviceAgent                      serviceAccount:service-430748831547@compute-system.iam.gserviceaccount.com
roles/editor                                    serviceAccount:430748831547-compute@developer.gserviceaccount.com
roles/owner                                     user:fardenghi@itba.edu.ar
```

Ni `lectora` ni `sin-acceso` figuran.
