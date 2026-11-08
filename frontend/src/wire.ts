import { drawguess } from "./drawing.pb.js";
import type { State, Stroke } from "./game";

export const PROTOBUF_PROTOCOL = "drawguess.pb.v1";
const { DrawingPacket, PacketType } = drawguess.wire.v1;
type ProtoStroke = drawguess.wire.v1.IStrokeChunk;
export interface IncomingFrame {
  type: string;
  state?: State;
  strokes?: Stroke[];
  stroke?: Stroke;
  turn?: number;
  seq?: number;
  message?: string;
}
function fromProto(s: ProtoStroke): Stroke {
  const xy = s.xy || [];
  if (xy.length % 2 || xy.length > 48000)
    throw new Error("Invalid coordinate array");
  const points = [];
  for (let i = 0; i < xy.length; i += 2) {
    const x = xy[i]!,
      y = xy[i + 1]!;
    if (x > 10000 || y > 10000) throw new Error("Coordinate out of range");
    points.push({ x, y });
  }
  return {
    id: s.id || "",
    color: s.color || "",
    width: s.width || 0,
    tool: s.eraser ? "eraser" : "pen",
    points,
  };
}
export function encodeStroke(turn: number, stroke: Stroke): Uint8Array {
  return DrawingPacket.encode({
    type: PacketType.STROKE,
    turn,
    stroke: {
      id: stroke.id,
      color: stroke.color,
      width: stroke.width,
      eraser: stroke.tool === "eraser",
      xy: stroke.points.flatMap((p) => [p.x, p.y]),
    },
  }).finish();
}
export function decodeDrawing(data: ArrayBuffer): IncomingFrame {
  const p = DrawingPacket.decode(new Uint8Array(data));
  if (p.type === PacketType.STROKE && p.stroke) {
    if ((p.stroke.xy?.length || 0) > 192)
      throw new Error("Stroke batch too large");
    return {
      type: "stroke",
      turn: p.turn,
      seq: p.seq,
      stroke: fromProto(p.stroke),
    };
  }
  if (p.type === PacketType.SNAPSHOT) {
    if (p.strokes.length > 600) throw new Error("Too many strokes");
    const strokes = p.strokes.map(fromProto);
    if (
      strokes.reduce((count, stroke) => count + stroke.points.length, 0) > 24000
    )
      throw new Error("Too many points");
    return {
      type: "snapshot",
      turn: p.turn,
      seq: p.seq,
      strokes,
      state: JSON.parse(new TextDecoder().decode(p.stateJson)),
    };
  }
  throw new Error("Unsupported drawing packet");
}
