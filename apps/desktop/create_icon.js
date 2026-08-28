const fs = require('fs');
const path = require('path');
const zlib = require('zlib');

// Create a 256x256 PNG image with a teal shield icon
function createIconPng(width = 256, height = 256) {
  // Raw RGBA pixel buffer
  const rawData = Buffer.alloc((width * 4 + 1) * height);

  for (let y = 0; y < height; y++) {
    const rowOffset = y * (width * 4 + 1);
    rawData[rowOffset] = 0; // Filter type: None

    for (let x = 0; x < width; x++) {
      const pxOffset = rowOffset + 1 + x * 4;

      // Center distance
      const dx = x - width / 2;
      const dy = y - height / 2;
      const dist = Math.sqrt(dx * dx + dy * dy);

      const inBox = Math.abs(dx) <= width * 0.4 && Math.abs(dy) <= height * 0.4;
      const inCircle = dist <= width * 0.45;

      if (inCircle && inBox) {
        // Teal background (#0f766e)
        const inInnerShield = Math.abs(dx) <= width * 0.2 && (dy >= -height * 0.15 && dy <= height * 0.2);
        if (inInnerShield && (Math.abs(dx) <= width * 0.06 || Math.abs(dy) <= height * 0.06)) {
          // White symbol (#ffffff)
          rawData[pxOffset] = 255;
          rawData[pxOffset + 1] = 255;
          rawData[pxOffset + 2] = 255;
          rawData[pxOffset + 3] = 255;
        } else {
          rawData[pxOffset] = 15;   // R: 15
          rawData[pxOffset + 1] = 118; // G: 118
          rawData[pxOffset + 2] = 110; // B: 110
          rawData[pxOffset + 3] = 255; // Alpha
        }
      } else {
        // Transparent
        rawData[pxOffset] = 0;
        rawData[pxOffset + 1] = 0;
        rawData[pxOffset + 2] = 0;
        rawData[pxOffset + 3] = 0;
      }
    }
  }

  // Compress IDAT
  const compressed = zlib.deflateSync(rawData);

  // PNG Header
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);

  function chunk(type, data) {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length, 0);

    const typeBuf = Buffer.from(type, 'ascii');
    const body = Buffer.concat([typeBuf, data]);

    const crcBuf = Buffer.alloc(4);
    crcBuf.writeUInt32BE(crc32(body), 0);

    return Buffer.concat([len, body, crcBuf]);
  }

  function crc32(buf) {
    let crc = 0xffffffff;
    for (let i = 0; i < buf.length; i++) {
      crc = crcTable[(crc ^ buf[i]) & 0xff] ^ (crc >>> 8);
    }
    return (crc ^ 0xffffffff) >>> 0;
  }

  const crcTable = new Uint32Array(256);
  for (let i = 0; i < 256; i++) {
    let c = i;
    for (let k = 0; k < 8; k++) {
      c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    }
    crcTable[i] = c;
  }

  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 6;
  ihdr[10] = 0;
  ihdr[11] = 0;
  ihdr[12] = 0;

  const ihdrChunk = chunk('IHDR', ihdr);
  const idatChunk = chunk('IDAT', compressed);
  const iendChunk = chunk('IEND', Buffer.alloc(0));

  return Buffer.concat([signature, ihdrChunk, idatChunk, iendChunk]);
}

function pngToIco(pngBuffer) {
  // 6-byte header
  const header = Buffer.alloc(6);
  header.writeUInt16LE(0, 0); // Reserved
  header.writeUInt16LE(1, 2); // Type 1 = ICO
  header.writeUInt16LE(1, 4); // Count = 1 image

  // 16-byte Directory entry
  const entry = Buffer.alloc(16);
  entry.writeUInt8(0, 0); // Width 256 is represented by 0
  entry.writeUInt8(0, 1); // Height 256 is represented by 0
  entry.writeUInt8(0, 2); // Colors = 0
  entry.writeUInt8(0, 3); // Reserved
  entry.writeUInt16LE(1, 4); // Color planes
  entry.writeUInt16LE(32, 6); // Bits per pixel
  entry.writeUInt32LE(pngBuffer.length, 8); // Size of image data
  entry.writeUInt32LE(22, 12); // Offset: 6 (header) + 16 (entry) = 22

  return Buffer.concat([header, entry, pngBuffer]);
}

const outDir = path.join(__dirname, 'build');
if (!fs.existsSync(outDir)) {
  fs.mkdirSync(outDir, { recursive: true });
}

const png = createIconPng(256, 256);
fs.writeFileSync(path.join(outDir, 'icon.png'), png);
console.log('Created 256x256 icon.png');

const ico = pngToIco(png);
fs.writeFileSync(path.join(outDir, 'icon.ico'), ico);
console.log('Created valid Windows icon.ico');
