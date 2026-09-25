import { Upload } from "lucide-react";
import { toast } from "sonner";

import { useI18n } from "@/i18n";
import { useDelivery, usePublish } from "@/hooks/usePublish";
import { DeliveryStages, PublishProblems } from "@/components/publish/Delivery";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";

interface Props {
  ids: string[];
  /** paths names files that are not items, such as kite.yaml. */
  paths?: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDone: () => void;
  title?: string;
  description?: string;
}

/** Publishing several items at once, with the reasons it might be refused. */
export function PublishDialog({ ids, paths, open, onOpenChange, onDone, title, description }: Props) {
  const { t } = useI18n();
  const delivery = useDelivery();
  const publish = usePublish(
    ids,
    (result) => {
      toast.success(t(result.rebased ? "publish.rebased" : "publish.done"));
      onOpenChange(false);
      onDone();
    },
    undefined,
    paths,
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) publish.reset();
        onOpenChange(next);
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title ?? t("publish.selectedTitle", { count: ids.length })}</DialogTitle>
          <DialogDescription>{description ?? t("publish.selectedNote")}</DialogDescription>
        </DialogHeader>

        <DeliveryStages delivery={delivery.data} publish={publish} />
        <PublishProblems publish={publish} />

        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">{t("common.cancel")}</Button>
          </DialogClose>
          <Button disabled={publish.pending} onClick={() => void publish.run()}>
            {publish.pending ? (
              <Spinner />
            ) : (
              <Upload />
            )}
            {publish.needsConfirmation ? t("publish.anyway") : t("publish.action")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
