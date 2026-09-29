import { toast } from "sonner";

import type { Media } from "@/api/client";
import type { Key } from "@/i18n";

/**
 * stored hands back the link to an uploaded file, and tells the author when
 * the server took out where a photo was taken, which it does before storing
 * a photo that says. One notice stands for a batch of photos.
 */
export function stored(media: Media, t: (key: Key) => string): string {
  if (media.location_removed) toast.info(t("media.locationRemoved"), { id: "location-removed" });
  return media.link;
}
