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
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDone: () => void;
}

/** Publishing several items at once, with the reasons it might be refused. */
export function PublishDialog({ ids, open, onOpenChange, onDone }: Props) {
  const { t } = useI18n();
  const delivery = useDelivery();
  const publish = usePublish(ids, () => {
    toast.success(t("publish.done"));
    onOpenChange(false);
    onDone();
  });

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
          <DialogTitle>{t("publish.selectedTitle", { count: ids.length })}</DialogTitle>
          <DialogDescription>{t("publish.selectedNote")}</DialogDescription>
        </DialogHeader>

        <DeliveryStages delivery={delivery.data} />
        <PublishProblems publish={publish} />

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>{t("common.cancel")}</DialogClose>
          <Button disabled={publish.pending} onClick={() => void publish.run()}>
            {publish.pending ? (
              <Spinner data-icon="inline-start" />
            ) : (
              <Upload data-icon="inline-start" />
            )}
            {publish.needsConfirmation ? t("publish.anyway") : t("publish.action")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
