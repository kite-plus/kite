/** The side of the square a picture is stored at. */
export const avatarSize = 256;

/**
 * squarePicture crops a picture to its middle square and scales it to
 * avatarSize, so what is sent is a few kilobytes whatever the camera took.
 *
 * WebP where the browser can write it, PNG where it cannot; the server keeps
 * either. An animated GIF keeps its first frame.
 */
export async function squarePicture(file: Blob): Promise<Blob> {
  const bitmap = await createImageBitmap(file);
  try {
    const side = Math.min(bitmap.width, bitmap.height);
    const canvas = document.createElement("canvas");
    canvas.width = canvas.height = Math.min(avatarSize, side);

    const context = canvas.getContext("2d");
    if (!context) throw new Error("no 2d canvas");
    context.imageSmoothingQuality = "high";
    context.drawImage(
      bitmap,
      (bitmap.width - side) / 2,
      (bitmap.height - side) / 2,
      side,
      side,
      0,
      0,
      canvas.width,
      canvas.height,
    );

    return await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (blob) => (blob ? resolve(blob) : reject(new Error("the picture could not be encoded"))),
        "image/webp",
        0.9,
      ),
    );
  } finally {
    bitmap.close();
  }
}
