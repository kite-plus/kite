import { useMutation } from "@tanstack/react-query";

import { ApiError, api } from "@/api/client";

/** An exported site: the archive, and the name the server gave it. */
export interface Exported {
  archive: Blob;
  name: string;
}

/**
 * useExportSite builds the site as a deployment gets it and fetches it as a
 * zip archive. The build can take a while, so it is a mutation the page can
 * show progress for, and saving it is a separate step.
 */
export function useExportSite() {
  return useMutation({
    mutationFn: async (): Promise<Exported> => {
      const { data, error, response } = await api.POST("/export", { parseAs: "blob" });
      if (error) throw new ApiError(error.error.code, error.error.message, error.error.field);
      return {
        archive: data as unknown as Blob,
        name: attachmentName(response.headers.get("Content-Disposition")) ?? "site.zip",
      };
    },
  });
}

/** attachmentName reads the file name out of a Content-Disposition header. */
function attachmentName(header: string | null): string | undefined {
  const match = header?.match(/filename\*?=(?:UTF-8'')?"?([^";]+)"?/i);
  return match ? decodeURIComponent(match[1]) : undefined;
}

/** saveFile hands a file to the browser to save, as a download link would. */
export function saveFile(file: Blob, name: string) {
  const url = URL.createObjectURL(file);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.append(link);
  link.click();
  link.remove();
  // Revoked after the click has been handled, not during it.
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
