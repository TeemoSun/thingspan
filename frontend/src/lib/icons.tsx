import type { ComponentType } from "react";
import {
  IconAirConditioning,
  IconArmchair,
  IconBed,
  IconBike,
  IconBooks,
  IconBottle,
  IconCake,
  IconCamera,
  IconCar,
  IconCoffee,
  IconCreditCard,
  IconCrown,
  IconDeviceDesktop,
  IconDeviceGamepad2,
  IconDeviceLaptop,
  IconDeviceMobile,
  IconDiamond,
  IconDumbbell,
  IconFridge,
  IconGift,
  IconHeadphones,
  IconHeart,
  IconKeyboard,
  IconLeaf,
  IconMicrophone,
  IconMusic,
  IconPackage,
  IconPhone,
  IconPiano,
  IconPlane,
  IconShirt,
  IconShoe,
  IconSofa,
  IconTent,
  IconTicket,
  IconTools,
  IconWallet,
} from "@tabler/icons-react";

export interface AssetIconOption {
  name: string;
  label: string;
  icon: ComponentType<{ className?: string; size?: number | string }>;
}

export const ASSET_ICONS: AssetIconOption[] = [
  { name: "package", label: "默认", icon: IconPackage },
  { name: "device-mobile", label: "手机", icon: IconDeviceMobile },
  { name: "phone", label: "电话", icon: IconPhone },
  { name: "device-laptop", label: "笔记本", icon: IconDeviceLaptop },
  { name: "device-desktop", label: "台式机", icon: IconDeviceDesktop },
  { name: "keyboard", label: "键鼠", icon: IconKeyboard },
  { name: "camera", label: "相机", icon: IconCamera },
  { name: "headphones", label: "耳机", icon: IconHeadphones },
  { name: "microphone", label: "麦克风", icon: IconMicrophone },
  { name: "device-gamepad-2", label: "游戏机", icon: IconDeviceGamepad2 },
  { name: "fridge", label: "冰箱", icon: IconFridge },
  { name: "air-conditioning", label: "空调", icon: IconAirConditioning },
  { name: "sofa", label: "沙发", icon: IconSofa },
  { name: "bed", label: "床", icon: IconBed },
  { name: "tools", label: "工具", icon: IconTools },
  { name: "bike", label: "自行车", icon: IconBike },
  { name: "car", label: "汽车", icon: IconCar },
  { name: "plane", label: "飞机", icon: IconPlane },
  { name: "books", label: "图书", icon: IconBooks },
  { name: "shirt", label: "衬衫", icon: IconShirt },
  { name: "shoe", label: "鞋", icon: IconShoe },
  { name: "wallet", label: "钱包", icon: IconWallet },
  { name: "credit-card", label: "银行卡", icon: IconCreditCard },
  { name: "ticket", label: "票据", icon: IconTicket },
  { name: "crown", label: "会员", icon: IconCrown },
  { name: "coffee", label: "咖啡", icon: IconCoffee },
  { name: "cake", label: "蛋糕", icon: IconCake },
  { name: "gift", label: "礼物", icon: IconGift },
  { name: "heart", label: "心形", icon: IconHeart },
  { name: "diamond", label: "钻石", icon: IconDiamond },
  { name: "bottle", label: "瓶子", icon: IconBottle },
  { name: "leaf", label: "植物", icon: IconLeaf },
  { name: "music", label: "音乐", icon: IconMusic },
  { name: "piano", label: "钢琴", icon: IconPiano },
  { name: "armchair", label: "扶手椅", icon: IconArmchair },
  { name: "dumbbell", label: "哑铃", icon: IconDumbbell },
  { name: "tent", label: "帐篷", icon: IconTent },
];

export function AssetIcon({
  name,
  className,
  size = 16,
}: {
  name?: string | null;
  className?: string;
  size?: number;
}) {
  const option = ASSET_ICONS.find((o) => o.name === name);
  const Icon = option?.icon ?? IconPackage;
  return <Icon className={className} size={size} />;
}
