import { useEffect, useState, type ComponentType } from "react";
import { IconPackage } from "@tabler/icons-react";

export type IconComponent = ComponentType<{ className?: string; size?: number | string }>;

let iconsPromise: Promise<Record<string, IconComponent>> | null = null;

export function loadIconModule(): Promise<Record<string, IconComponent>> {
  iconsPromise ??= import("@tabler/icons-react").then(
    (m) => m as unknown as Record<string, IconComponent>
  );
  return iconsPromise;
}

export function iconComponentName(slug: string): string {
  return (
    "Icon" +
    slug
      .split("-")
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join("")
  );
}

export function AssetIcon({
  name,
  className,
  size = 16,
}: {
  name?: string | null;
  className?: string;
  size?: number;
}) {
  const [Icon, setIcon] = useState<IconComponent | null>(null);

  useEffect(() => {
    let alive = true;
    if (!name) {
      setIcon(IconPackage);
      return;
    }
    loadIconModule().then((mod) => {
      if (alive) setIcon(mod[iconComponentName(name)] ?? IconPackage);
    });
    return () => {
      alive = false;
    };
  }, [name]);

  if (!Icon) return null;
  return <Icon className={className} size={size} />;
}

const FAV_KEY = "thingspan_icon_favs";
const RECENT_KEY = "thingspan_icon_recent";
const RECENT_LIMIT = 20;

function readList(key: string): string[] {
  try {
    return JSON.parse(localStorage.getItem(key) ?? "[]");
  } catch {
    return [];
  }
}

function writeList(key: string, list: string[]) {
  localStorage.setItem(key, JSON.stringify(list));
}

export function loadFavIcons(): string[] {
  return readList(FAV_KEY);
}

export function toggleFavIcon(slug: string): string[] {
  const favs = readList(FAV_KEY);
  const next = favs.includes(slug) ? favs.filter((s) => s !== slug) : [...favs, slug];
  writeList(FAV_KEY, next);
  return next;
}

export function loadRecentIcons(): string[] {
  return readList(RECENT_KEY);
}

export function pushRecentIcon(slug: string): string[] {
  const recent = readList(RECENT_KEY).filter((s) => s !== slug);
  recent.unshift(slug);
  writeList(RECENT_KEY, recent.slice(0, RECENT_LIMIT));
  return recent.slice(0, RECENT_LIMIT);
}
