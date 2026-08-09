import { Badge } from "@/components/ui/badge";
import { STATUS_LABELS, type AssetStatus } from "@/lib/types";

const STATUS_VARIANT: Record<AssetStatus, "success" | "secondary" | "destructive" | "warning"> = {
  in_use: "success",
  sold: "secondary",
  broken: "destructive",
  expired: "warning",
};

export function StatusBadge({ status }: { status: AssetStatus }) {
  return <Badge variant={STATUS_VARIANT[status]}>{STATUS_LABELS[status]}</Badge>;
}
