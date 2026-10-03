import { Trash2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

/**
 * RemoveButton is a card's remove action. While it cannot be used, blocked
 * says why on hover, which a disabled button cannot say for itself.
 */
export function RemoveButton({
  label,
  blocked,
  onClick,
  className,
}: {
  label: string;
  blocked?: string;
  onClick: () => void;
  className?: string;
}) {
  const button = (
    <Button
      variant="ghost"
      size="sm"
      disabled={blocked !== undefined}
      onClick={onClick}
      className="text-destructive hover:bg-destructive/10 hover:text-destructive"
    >
      <Trash2 />
      {label}
    </Button>
  );
  if (blocked === undefined) return <span className={cn("inline-flex", className)}>{button}</span>;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0} className={cn("inline-flex rounded-md", className)}>
          {button}
        </span>
      </TooltipTrigger>
      <TooltipContent>{blocked}</TooltipContent>
    </Tooltip>
  );
}
