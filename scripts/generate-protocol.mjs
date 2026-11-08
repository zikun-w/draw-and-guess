import { existsSync } from "node:fs";
import { delimiter, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const frontend = join(root, "frontend");
const localCompiler = join(
  root,
  ".tools",
  "protoc",
  "bin",
  process.platform === "win32" ? "protoc.exe" : "protoc",
);
const compiler =
  process.env.PROTOC || (existsSync(localCompiler) ? localCompiler : "protoc");
const env = {
  ...process.env,
  PATH: join(root, ".tools", "bin") + delimiter + process.env.PATH,
};
function run(program, args) {
  const result = spawnSync(program, args, {
    cwd: root,
    env,
    stdio: "inherit",
    shell: false,
  });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status || 1);
}
run(compiler, [
  `--proto_path=${join(root, "protocol")}`,
  `--go_out=${join(root, "backend")}`,
  "--go_opt=module=drawguess",
  join(root, "protocol", "drawing.proto"),
]);
const generated = join(frontend, "src", "drawing.pb.js");
run(process.execPath, [
  join(frontend, "node_modules", "protobufjs-cli", "bin", "pbjs"),
  "--target",
  "static-module",
  "--wrap",
  "es6",
  "--no-verify",
  "--no-convert",
  "--no-delimited",
  "--no-service",
  "--out",
  generated,
  join(root, "protocol", "drawing.proto"),
]);
run(process.execPath, [
  join(frontend, "node_modules", "protobufjs-cli", "bin", "pbts"),
  "--out",
  join(frontend, "src", "drawing.pb.d.ts"),
  generated,
]);
