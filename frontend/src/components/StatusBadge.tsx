import { Badge } from "@/components/ui/badge";
import { STATUS_LABELS, type AssetStatus } from "@/lib/types";

const STATUS_VARIANT: Record<AssetStatus, "success" | "secondary" | "destructive" | "warning"> = {
  in_use: "success",
  sold: "secondary",
  broken: "destructive",
  expired: "warning",
};

export function StatusBadge({ status, className }: { status: AssetStatus; className?: string }) {
  const variant = STATUS_VARIANT[status] ?? "secondary";
  const label = STATUS_LABELS[status] ?? status;
  return <Badge variant={variant} className={className}>{label}</Badge>;
}

