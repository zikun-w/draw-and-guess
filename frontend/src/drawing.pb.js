/*eslint-disable block-scoped-var, id-length, no-control-regex, no-magic-numbers, no-mixed-operators, no-prototype-builtins, no-redeclare, no-shadow, no-var, sort-vars, default-case, jsdoc/require-param*/
import $protobuf from "protobufjs/minimal.js";

// Common aliases
const $Reader = $protobuf.Reader, $Writer = $protobuf.Writer, $util = $protobuf.util;
const $Object = $util.global.Object, $undefined = $util.global.undefined, $Error = $util.global.Error, $RangeError = $util.global.RangeError;

// Exported root namespace
const $root = $protobuf.roots["default"] || ($protobuf.roots["default"] = {});

export const drawguess = $root.drawguess = (() => {

    /**
     * Namespace drawguess.
     * @exports drawguess
     * @namespace
     */
    const drawguess = {};

    drawguess.wire = (function() {

        /**
         * Namespace wire.
         * @memberof drawguess
         * @namespace
         */
        const wire = {};

        wire.v1 = (function() {

            /**
             * Namespace v1.
             * @memberof drawguess.wire
             * @namespace
             */
            const v1 = {};

            /**
             * PacketType enum.
             * @name drawguess.wire.v1.PacketType
             * @enum {number}
             * @property {number} PACKET_UNSPECIFIED=0 PACKET_UNSPECIFIED value
             * @property {number} STROKE=1 STROKE value
             * @property {number} SNAPSHOT=2 SNAPSHOT value
             */
            v1.PacketType = (function() {
                const valuesById = $Object.create(null), values = $Object.create(valuesById);
                values[valuesById[0] = "PACKET_UNSPECIFIED"] = 0;
                values[valuesById[1] = "STROKE"] = 1;
                values[valuesById[2] = "SNAPSHOT"] = 2;
                return values;
            })();

            v1.StrokeChunk = (function() {

                /**
                 * Properties of a StrokeChunk.
                 * @typedef {Object} drawguess.wire.v1.StrokeChunk.$Properties
                 * @property {string|null} [id] StrokeChunk id
                 * @property {string|null} [color] StrokeChunk color
                 * @property {number|null} [width] StrokeChunk width
                 * @property {boolean|null} [eraser] StrokeChunk eraser
                 * @property {Array.<number>|null} [xy] StrokeChunk xy
                 * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
                 */

                /**
                 * Properties of a StrokeChunk.
                 * @memberof drawguess.wire.v1
                 * @interface IStrokeChunk
                 * @augments drawguess.wire.v1.StrokeChunk.$Properties
                 * @deprecated Use drawguess.wire.v1.StrokeChunk.$Properties instead.
                 */

                /**
                 * Shape of a StrokeChunk.
                 * @typedef {drawguess.wire.v1.StrokeChunk.$Properties} drawguess.wire.v1.StrokeChunk.$Shape
                 */

                /**
                 * Constructs a new StrokeChunk.
                 * @memberof drawguess.wire.v1
                 * @classdesc Represents a StrokeChunk.
                 * @constructor
                 * @param {drawguess.wire.v1.StrokeChunk.$Properties=} [properties] Properties to set
                 * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
                 */
                const StrokeChunk = function (properties) {
                    this.xy = [];
                    if (properties)
                        for (let keys = $Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null && keys[i] !== "__proto__")
                                this[keys[i]] = properties[keys[i]];
                };

                /**
                 * StrokeChunk id.
                 * @member {string} id
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @instance
                 */
                StrokeChunk.prototype.id = "";

                /**
                 * StrokeChunk color.
                 * @member {string} color
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @instance
                 */
                StrokeChunk.prototype.color = "";

                /**
                 * StrokeChunk width.
                 * @member {number} width
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @instance
                 */
                StrokeChunk.prototype.width = 0;

                /**
                 * StrokeChunk eraser.
                 * @member {boolean} eraser
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @instance
                 */
                StrokeChunk.prototype.eraser = false;

                /**
                 * StrokeChunk xy.
                 * @member {Array.<number>} xy
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @instance
                 */
                StrokeChunk.prototype.xy = $util.emptyArray;

                /**
                 * Creates a new StrokeChunk instance using the specified properties.
                 * @function create
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @static
                 * @param {drawguess.wire.v1.StrokeChunk.$Properties=} [properties] Properties to set
                 * @returns {drawguess.wire.v1.StrokeChunk} StrokeChunk instance
                 * @type {{
                 *   (properties: drawguess.wire.v1.StrokeChunk.$Shape): drawguess.wire.v1.StrokeChunk & drawguess.wire.v1.StrokeChunk.$Shape;
                 *   (properties?: drawguess.wire.v1.StrokeChunk.$Properties): drawguess.wire.v1.StrokeChunk;
                 * }}
                 */
                StrokeChunk.create = function(properties) {
                    return new StrokeChunk(properties);
                };

                /**
                 * Encodes the specified StrokeChunk message. Does not implicitly {@link drawguess.wire.v1.StrokeChunk.verify|verify} messages.
                 * @function encode
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @static
                 * @param {drawguess.wire.v1.StrokeChunk.$Properties} message StrokeChunk message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                StrokeChunk.encode = function (message, writer, _depth) {
                    if (!writer)
                        writer = $Writer.create();
                    if (_depth === $undefined)
                        _depth = 0;
                    if (_depth > $util.recursionLimit)
                        throw $Error("max depth exceeded");
                    if (message.id != null && $Object.hasOwnProperty.call(message, "id") && message.id !== "")
                        writer.uint32(/* id 1, wireType 2 =*/10).string(message.id);
                    if (message.color != null && $Object.hasOwnProperty.call(message, "color") && message.color !== "")
                        writer.uint32(/* id 2, wireType 2 =*/18).string(message.color);
                    if (message.width != null && $Object.hasOwnProperty.call(message, "width") && message.width !== 0)
                        writer.uint32(/* id 3, wireType 0 =*/24).uint32(message.width);
                    if (message.eraser != null && $Object.hasOwnProperty.call(message, "eraser") && message.eraser !== false)
                        writer.uint32(/* id 4, wireType 0 =*/32).bool(message.eraser);
                    if (message.xy != null && message.xy.length)
                        writer.uint32(/* id 5, wireType 2 =*/42).uint32s(message.xy);
                    if (message.$unknowns != null && $Object.hasOwnProperty.call(message, "$unknowns"))
                        for (let i = 0; i < message.$unknowns.length; ++i)
                            writer.raw(message.$unknowns[i]);
                    return writer;
                };

                /**
                 * Decodes a StrokeChunk message from the specified reader or buffer.
                 * @function decode
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {drawguess.wire.v1.StrokeChunk & drawguess.wire.v1.StrokeChunk.$Shape} StrokeChunk
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                StrokeChunk.decode = function (reader, length, _end, _depth, _target) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    if (_depth === $undefined)
                        _depth = 0;
                    if (_depth > $Reader.recursionLimit)
                        throw $Error("max depth exceeded");
                    let end, message, value;
                    if (length === $undefined)
                        end = reader.len;
                    else {
                        end = reader.pos + length;
                        if (end > reader.len)
                            throw $RangeError("index out of range");
                        length = reader.len;
                        reader.len = end;
                    }
                    message = _target || new $root.drawguess.wire.v1.StrokeChunk();
                    while (reader.pos < end) {
                        let start = reader.pos;
                        let tag = reader.tag();
                        if (tag === _end) {
                            _end = $undefined;
                            break;
                        }
                        let wireType = tag & 7;
                        switch (tag >>>= 3) {
                        case 1: {
                                if (wireType !== 2)
                                    break;
                                if ((value = reader.stringVerify()).length)
                                    message.id = value;
                                else
                                    delete message.id;
                                continue;
                            }
                        case 2: {
                                if (wireType !== 2)
                                    break;
                                if ((value = reader.stringVerify()).length)
                                    message.color = value;
                                else
                                    delete message.color;
                                continue;
                            }
                        case 3: {
                                if (wireType !== 0)
                                    break;
                                if (value = reader.uint32())
                                    message.width = value;
                                else
                                    delete message.width;
                                continue;
                            }
                        case 4: {
                                if (wireType !== 0)
                                    break;
                                if (value = reader.bool())
                                    message.eraser = value;
                                else
                                    delete message.eraser;
                                continue;
                            }
                        case 5: {
                                if (wireType === 2) {
                                    if (!(message.xy && message.xy.length))
                                        message.xy = [];
                                    reader.uint32s(message.xy);
                                    continue;
                                }
                                if (wireType !== 0)
                                    break;
                                if (!(message.xy && message.xy.length))
                                    message.xy = [];
                                message.xy.push(reader.uint32());
                                continue;
                            }
                        }
                        reader.skipType(wireType, _depth, tag);
                        if (!reader.discardUnknown) {
                            $util.makeProp(message, "$unknowns", false);
                            (message.$unknowns || (message.$unknowns = [])).push(reader.raw(start, reader.pos));
                        }
                    }
                    if (length !== $undefined) {
                        if (reader.pos !== end)
                            throw $RangeError("index out of range");
                        reader.len = length;
                    }
                    if (_end !== $undefined)
                        throw $Error("missing end group");
                    return message;
                };

                /**
                 * Gets the type url for StrokeChunk
                 * @function getTypeUrl
                 * @memberof drawguess.wire.v1.StrokeChunk
                 * @static
                 * @param {string} [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
                 * @returns {string} The type url
                 */
                StrokeChunk.getTypeUrl = function(prefix) {
                    if (prefix === $undefined)
                        prefix = "type.googleapis.com";
                    return prefix + "/drawguess.wire.v1.StrokeChunk";
                };

                return StrokeChunk;
            })();

            v1.DrawingPacket = (function() {

                /**
                 * Properties of a DrawingPacket.
                 * @typedef {Object} drawguess.wire.v1.DrawingPacket.$Properties
                 * @property {drawguess.wire.v1.PacketType|null} [type] DrawingPacket type
                 * @property {number|null} [turn] DrawingPacket turn
                 * @property {number|null} [seq] DrawingPacket seq
                 * @property {drawguess.wire.v1.StrokeChunk.$Properties|null} [stroke] DrawingPacket stroke
                 * @property {Array.<drawguess.wire.v1.StrokeChunk.$Properties>|null} [strokes] DrawingPacket strokes
                 * @property {Uint8Array|null} [stateJson] DrawingPacket stateJson
                 * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
                 */

                /**
                 * Properties of a DrawingPacket.
                 * @memberof drawguess.wire.v1
                 * @interface IDrawingPacket
                 * @augments drawguess.wire.v1.DrawingPacket.$Properties
                 * @deprecated Use drawguess.wire.v1.DrawingPacket.$Properties instead.
                 */

                /**
                 * Shape of a DrawingPacket.
                 * @typedef {drawguess.wire.v1.DrawingPacket.$Properties} drawguess.wire.v1.DrawingPacket.$Shape
                 */

                /**
                 * Constructs a new DrawingPacket.
                 * @memberof drawguess.wire.v1
                 * @classdesc Represents a DrawingPacket.
                 * @constructor
                 * @param {drawguess.wire.v1.DrawingPacket.$Properties=} [properties] Properties to set
                 * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
                 */
                const DrawingPacket = function (properties) {
                    this.strokes = [];
                    if (properties)
                        for (let keys = $Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null && keys[i] !== "__proto__")
                                this[keys[i]] = properties[keys[i]];
                };

                /**
                 * DrawingPacket type.
                 * @member {drawguess.wire.v1.PacketType} type
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @instance
                 */
                DrawingPacket.prototype.type = 0;

                /**
                 * DrawingPacket turn.
                 * @member {number} turn
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @instance
                 */
                DrawingPacket.prototype.turn = 0;

                /**
                 * DrawingPacket seq.
                 * @member {number} seq
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @instance
                 */
                DrawingPacket.prototype.seq = 0;

                /**
                 * DrawingPacket stroke.
                 * @member {drawguess.wire.v1.StrokeChunk.$Properties|null|undefined} stroke
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @instance
                 */
                DrawingPacket.prototype.stroke = null;

                /**
                 * DrawingPacket strokes.
                 * @member {Array.<drawguess.wire.v1.StrokeChunk.$Properties>} strokes
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @instance
                 */
                DrawingPacket.prototype.strokes = $util.emptyArray;

                /**
                 * DrawingPacket stateJson.
                 * @member {Uint8Array} stateJson
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @instance
                 */
                DrawingPacket.prototype.stateJson = $util.newBuffer([]);

                /**
                 * Creates a new DrawingPacket instance using the specified properties.
                 * @function create
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @static
                 * @param {drawguess.wire.v1.DrawingPacket.$Properties=} [properties] Properties to set
                 * @returns {drawguess.wire.v1.DrawingPacket} DrawingPacket instance
                 * @type {{
                 *   (properties: drawguess.wire.v1.DrawingPacket.$Shape): drawguess.wire.v1.DrawingPacket & drawguess.wire.v1.DrawingPacket.$Shape;
                 *   (properties?: drawguess.wire.v1.DrawingPacket.$Properties): drawguess.wire.v1.DrawingPacket;
                 * }}
                 */
                DrawingPacket.create = function(properties) {
                    return new DrawingPacket(properties);
                };

                /**
                 * Encodes the specified DrawingPacket message. Does not implicitly {@link drawguess.wire.v1.DrawingPacket.verify|verify} messages.
                 * @function encode
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @static
                 * @param {drawguess.wire.v1.DrawingPacket.$Properties} message DrawingPacket message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                DrawingPacket.encode = function (message, writer, _depth) {
                    if (!writer)
                        writer = $Writer.create();
                    if (_depth === $undefined)
                        _depth = 0;
                    if (_depth > $util.recursionLimit)
                        throw $Error("max depth exceeded");
                    if (message.type != null && $Object.hasOwnProperty.call(message, "type") && message.type !== 0)
                        writer.uint32(/* id 1, wireType 0 =*/8).int32(message.type);
                    if (message.turn != null && $Object.hasOwnProperty.call(message, "turn") && message.turn !== 0)
                        writer.uint32(/* id 2, wireType 0 =*/16).uint32(message.turn);
                    if (message.seq != null && $Object.hasOwnProperty.call(message, "seq") && message.seq !== 0)
                        writer.uint32(/* id 3, wireType 0 =*/24).uint32(message.seq);
                    if (message.stroke != null && $Object.hasOwnProperty.call(message, "stroke"))
                        $root.drawguess.wire.v1.StrokeChunk.encode(message.stroke, writer.uint32(/* id 4, wireType 2 =*/34).fork(), _depth + 1).ldelim();
                    if (message.strokes != null && message.strokes.length)
                        for (let i = 0; i < message.strokes.length; ++i)
                            $root.drawguess.wire.v1.StrokeChunk.encode(message.strokes[i], writer.uint32(/* id 5, wireType 2 =*/42).fork(), _depth + 1).ldelim();
                    if (message.stateJson != null && $Object.hasOwnProperty.call(message, "stateJson") && message.stateJson.length)
                        writer.uint32(/* id 6, wireType 2 =*/50).bytes(message.stateJson);
                    if (message.$unknowns != null && $Object.hasOwnProperty.call(message, "$unknowns"))
                        for (let i = 0; i < message.$unknowns.length; ++i)
                            writer.raw(message.$unknowns[i]);
                    return writer;
                };

                /**
                 * Decodes a DrawingPacket message from the specified reader or buffer.
                 * @function decode
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {drawguess.wire.v1.DrawingPacket & drawguess.wire.v1.DrawingPacket.$Shape} DrawingPacket
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                DrawingPacket.decode = function (reader, length, _end, _depth, _target) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    if (_depth === $undefined)
                        _depth = 0;
                    if (_depth > $Reader.recursionLimit)
                        throw $Error("max depth exceeded");
                    let end, message, value;
                    if (length === $undefined)
                        end = reader.len;
                    else {
                        end = reader.pos + length;
                        if (end > reader.len)
                            throw $RangeError("index out of range");
                        length = reader.len;
                        reader.len = end;
                    }
                    message = _target || new $root.drawguess.wire.v1.DrawingPacket();
                    while (reader.pos < end) {
                        let start = reader.pos;
                        let tag = reader.tag();
                        if (tag === _end) {
                            _end = $undefined;
                            break;
                        }
                        let wireType = tag & 7;
                        switch (tag >>>= 3) {
                        case 1: {
                                if (wireType !== 0)
                                    break;
                                if (value = reader.int32())
                                    message.type = value;
                                else
                                    delete message.type;
                                continue;
                            }
                        case 2: {
                                if (wireType !== 0)
                                    break;
                                if (value = reader.uint32())
                                    message.turn = value;
                                else
                                    delete message.turn;
                                continue;
                            }
                        case 3: {
                                if (wireType !== 0)
                                    break;
                                if (value = reader.uint32())
                                    message.seq = value;
                                else
                                    delete message.seq;
                                continue;
                            }
                        case 4: {
                                if (wireType !== 2)
                                    break;
                                message.stroke = $root.drawguess.wire.v1.StrokeChunk.decode(reader, reader.size(), $undefined, _depth + 1, message.stroke);
                                continue;
                            }
                        case 5: {
                                if (wireType !== 2)
                                    break;
                                if (!(message.strokes && message.strokes.length))
                                    message.strokes = [];
                                message.strokes.push($root.drawguess.wire.v1.StrokeChunk.decode(reader, reader.size(), $undefined, _depth + 1));
                                continue;
                            }
                        case 6: {
                                if (wireType !== 2)
                                    break;
                                if ((value = reader.bytes()).length)
                                    message.stateJson = value;
                                else
                                    delete message.stateJson;
                                continue;
                            }
                        }
                        reader.skipType(wireType, _depth, tag);
                        if (!reader.discardUnknown) {
                            $util.makeProp(message, "$unknowns", false);
                            (message.$unknowns || (message.$unknowns = [])).push(reader.raw(start, reader.pos));
                        }
                    }
                    if (length !== $undefined) {
                        if (reader.pos !== end)
                            throw $RangeError("index out of range");
                        reader.len = length;
                    }
                    if (_end !== $undefined)
                        throw $Error("missing end group");
                    return message;
                };

                /**
                 * Gets the type url for DrawingPacket
                 * @function getTypeUrl
                 * @memberof drawguess.wire.v1.DrawingPacket
                 * @static
                 * @param {string} [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
                 * @returns {string} The type url
                 */
                DrawingPacket.getTypeUrl = function(prefix) {
                    if (prefix === $undefined)
                        prefix = "type.googleapis.com";
                    return prefix + "/drawguess.wire.v1.DrawingPacket";
                };

                return DrawingPacket;
            })();

            return v1;
        })();

        return wire;
    })();

    return drawguess;
})();

export {
  $root as default
};
