import * as $protobuf from "protobufjs";
import Long = require("long");

/** Namespace drawguess. */
export namespace drawguess {

    /** Namespace wire. */
    namespace wire {

        /** Namespace v1. */
        namespace v1 {

            /** PacketType enum. */
            enum PacketType {

                /** PACKET_UNSPECIFIED value */
                PACKET_UNSPECIFIED = 0,

                /** STROKE value */
                STROKE = 1,

                /** SNAPSHOT value */
                SNAPSHOT = 2
            }

            /**
             * Properties of a StrokeChunk.
             * @deprecated Use drawguess.wire.v1.StrokeChunk.$Properties instead.
             */
            interface IStrokeChunk extends drawguess.wire.v1.StrokeChunk.$Properties {
            }

            /** Represents a StrokeChunk. */
            class StrokeChunk {

                /**
                 * Constructs a new StrokeChunk.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: drawguess.wire.v1.StrokeChunk.$Properties);

                /** Unknown fields preserved while decoding when enabled */
                $unknowns?: Uint8Array[];

                /** StrokeChunk id. */
                id: string;

                /** StrokeChunk color. */
                color: string;

                /** StrokeChunk width. */
                width: number;

                /** StrokeChunk eraser. */
                eraser: boolean;

                /** StrokeChunk xy. */
                xy: number[];

                /**
                 * Creates a new StrokeChunk instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns StrokeChunk instance
                 */
                static create(properties: drawguess.wire.v1.StrokeChunk.$Shape): drawguess.wire.v1.StrokeChunk & drawguess.wire.v1.StrokeChunk.$Shape;
                static create(properties?: drawguess.wire.v1.StrokeChunk.$Properties): drawguess.wire.v1.StrokeChunk;

                /**
                 * Encodes the specified StrokeChunk message. Does not implicitly {@link drawguess.wire.v1.StrokeChunk.verify|verify} messages.
                 * @param message StrokeChunk message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                static encode(message: drawguess.wire.v1.StrokeChunk.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a StrokeChunk message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns {drawguess.wire.v1.StrokeChunk & drawguess.wire.v1.StrokeChunk.$Shape} StrokeChunk
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): drawguess.wire.v1.StrokeChunk & drawguess.wire.v1.StrokeChunk.$Shape;

                /**
                 * Gets the type url for StrokeChunk
                 * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
                 * @returns The type url
                 */
                static getTypeUrl(prefix?: string): string;
            }

            namespace StrokeChunk {

                /** Properties of a StrokeChunk. */
                interface $Properties {

                    /** StrokeChunk id */
                    id?: (string|null);

                    /** StrokeChunk color */
                    color?: (string|null);

                    /** StrokeChunk width */
                    width?: (number|null);

                    /** StrokeChunk eraser */
                    eraser?: (boolean|null);

                    /** StrokeChunk xy */
                    xy?: (number[]|null);

                    /** Unknown fields preserved while decoding when enabled */
                    $unknowns?: Uint8Array[];
                }

                /** Shape of a StrokeChunk. */
                type $Shape = drawguess.wire.v1.StrokeChunk.$Properties;
            }

            /**
             * Properties of a DrawingPacket.
             * @deprecated Use drawguess.wire.v1.DrawingPacket.$Properties instead.
             */
            interface IDrawingPacket extends drawguess.wire.v1.DrawingPacket.$Properties {
            }

            /** Represents a DrawingPacket. */
            class DrawingPacket {

                /**
                 * Constructs a new DrawingPacket.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: drawguess.wire.v1.DrawingPacket.$Properties);

                /** Unknown fields preserved while decoding when enabled */
                $unknowns?: Uint8Array[];

                /** DrawingPacket type. */
                type: drawguess.wire.v1.PacketType;

                /** DrawingPacket turn. */
                turn: number;

                /** DrawingPacket seq. */
                seq: number;

                /** DrawingPacket stroke. */
                stroke?: (drawguess.wire.v1.StrokeChunk.$Properties|null);

                /** DrawingPacket strokes. */
                strokes: drawguess.wire.v1.StrokeChunk.$Properties[];

                /** DrawingPacket stateJson. */
                stateJson: Uint8Array;

                /**
                 * Creates a new DrawingPacket instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns DrawingPacket instance
                 */
                static create(properties: drawguess.wire.v1.DrawingPacket.$Shape): drawguess.wire.v1.DrawingPacket & drawguess.wire.v1.DrawingPacket.$Shape;
                static create(properties?: drawguess.wire.v1.DrawingPacket.$Properties): drawguess.wire.v1.DrawingPacket;

                /**
                 * Encodes the specified DrawingPacket message. Does not implicitly {@link drawguess.wire.v1.DrawingPacket.verify|verify} messages.
                 * @param message DrawingPacket message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                static encode(message: drawguess.wire.v1.DrawingPacket.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a DrawingPacket message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns {drawguess.wire.v1.DrawingPacket & drawguess.wire.v1.DrawingPacket.$Shape} DrawingPacket
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): drawguess.wire.v1.DrawingPacket & drawguess.wire.v1.DrawingPacket.$Shape;

                /**
                 * Gets the type url for DrawingPacket
                 * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
                 * @returns The type url
                 */
                static getTypeUrl(prefix?: string): string;
            }

            namespace DrawingPacket {

                /** Properties of a DrawingPacket. */
                interface $Properties {

                    /** DrawingPacket type */
                    type?: (drawguess.wire.v1.PacketType|null);

                    /** DrawingPacket turn */
                    turn?: (number|null);

                    /** DrawingPacket seq */
                    seq?: (number|null);

                    /** DrawingPacket stroke */
                    stroke?: (drawguess.wire.v1.StrokeChunk.$Properties|null);

                    /** DrawingPacket strokes */
                    strokes?: (drawguess.wire.v1.StrokeChunk.$Properties[]|null);

                    /** DrawingPacket stateJson */
                    stateJson?: (Uint8Array|null);

                    /** Unknown fields preserved while decoding when enabled */
                    $unknowns?: Uint8Array[];
                }

                /** Shape of a DrawingPacket. */
                type $Shape = drawguess.wire.v1.DrawingPacket.$Properties;
            }
        }
    }
}
