import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { IconStarFilled } from "@tabler/icons-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";
import {
  type IconComponent,
  iconComponentName,
  loadFavIcons,
  loadIconModule,
  loadRecentIcons,
  pushRecentIcon,
  toggleFavIcon,
} from "@/lib/icons";
import { cn } from "@/lib/utils";

type Tab = "fav" | "recent" | "all";

const PAGE_SIZE = 120;

const TABS: [Tab, string][] = [
  ["fav", "收藏"],
  ["recent", "最近使用"],
  ["all", "全部"],
];

export default function IconPicker({
  open,
  onOpenChange,
  value,
  onSelect,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: string | null;
  onSelect: (icon: string) => void;
}) {
  const { data } = useQuery({
    queryKey: ["icons"],
    queryFn: () => api<{ icons: string[] }>("/api/icons"),
  });
  const [tab, setTab] = useState<Tab>("all");
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);
  const [favs, setFavs] = useState<string[]>([]);
  const [recent, setRecent] = useState<string[]>([]);
  const [mod, setMod] = useState<Record<string, IconComponent> | null>(null);

  useEffect(() => {
    if (open) {
      setFavs(loadFavIcons());
      setRecent(loadRecentIcons());
      setPage(1);
      setSearch("");
      setDebouncedSearch("");
      setTab("all");
      loadIconModule().then(setMod);
    }
  }, [open]);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 200);
    return () => clearTimeout(timer);
  }, [search]);

  const query = debouncedSearch.trim().toLowerCase();

  const list = useMemo(() => {
    let icons: string[];
    if (tab === "fav") icons = favs;
    else if (tab === "recent") icons = recent;
    else icons = data?.icons ?? [];
    if (query) icons = icons.filter((s) => s.includes(query));
    return icons;
  }, [tab, favs, recent, data, query]);


  const visible = list.slice(0, page * PAGE_SIZE);
  const hasMore = visible.length < list.length;

  function select(slug: string) {
    setRecent(pushRecentIcon(slug));
    onSelect(slug);
    onOpenChange(false);
  }

  function toggleFav(slug: string) {
    setFavs(toggleFavIcon(slug));
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>选择图标</DialogTitle>
          <DialogDescription>左键选择图标，右键收藏 / 取消收藏</DialogDescription>
        </DialogHeader>
        <Input
          placeholder="搜索图标名称，如 camera、phone、car"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(1);
          }}
        />
        <div className="flex gap-1">
          {TABS.map(([t, label]) => (
            <Button
              key={t}
              type="button"
              size="sm"
              variant={tab === t ? "default" : "outline"}
              onClick={() => {
                setTab(t);
                setPage(1);
              }}
            >
              {label}
            </Button>
          ))}
        </div>
        {!data ? (
          <p className="py-8 text-center text-sm text-muted-foreground">加载图标中…</p>
        ) : list.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            {tab === "fav"
              ? "暂无收藏，右键图标即可收藏"
              : tab === "recent"
                ? "暂无最近使用"
                : "未找到匹配的图标"}
          </p>
        ) : (
          <div className="flex max-h-[420px] flex-wrap gap-1.5 overflow-y-auto pr-1">
            {visible.map((slug) => {
              const Cmp = mod?.[iconComponentName(slug)];
              if (!Cmp) return null;
              return (
                <button
                  key={slug}
                  type="button"
                  title={slug}
                  onClick={() => select(slug)}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    toggleFav(slug);
                  }}
                  className={cn(
                    "relative rounded-md border p-1.5 hover:bg-muted",
                    value === slug && "border-primary bg-primary/10"
                  )}
                >
                  <Cmp size={18} />
                  {favs.includes(slug) && (
                    <IconStarFilled className="absolute -right-1 -top-1 h-3 w-3 text-amber-500" />
                  )}
                </button>
              );
            })}
          </div>
        )}
        {hasMore && (
          <div className="flex justify-center">
            <Button type="button" variant="outline" size="sm" onClick={() => setPage((p) => p + 1)}>
              加载更多（已显示 {visible.length} / {list.length}）
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
