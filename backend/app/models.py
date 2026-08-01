import enum
from datetime import datetime

from sqlalchemy import DateTime, ForeignKey, Integer, String, Text, UniqueConstraint, func
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base


class Program(str, enum.Enum):
    FRC = "FRC"
    FTC = "FTC"
    BOTH = "BOTH"


class TrackMode(str, enum.Enum):
    SNP = "SNP"
    BLK = "BLK"


class TransactionType(str, enum.Enum):
    IN = "IN"
    OUT = "OUT"
    RETURN = "RETURN"
    ADJUST = "ADJUST"


class Location(Base):
    __tablename__ = "locations"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    code: Mapped[str] = mapped_column(String(32), unique=True, index=True)
    name_zh: Mapped[str] = mapped_column(String(128), default="")
    name_en: Mapped[str] = mapped_column(String(128), default="")


class Operator(Base):
    __tablename__ = "operators"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    display_name: Mapped[str] = mapped_column(String(64), unique=True)
    active: Mapped[bool] = mapped_column(default=True)


class Item(Base):
    __tablename__ = "items"
    __table_args__ = (UniqueConstraint("barcode", name="uq_items_barcode"),)

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    barcode: Mapped[str] = mapped_column(String(48), index=True)
    program: Mapped[str] = mapped_column(String(8), default=Program.BOTH.value)
    track_mode: Mapped[str] = mapped_column(String(8), default=TrackMode.BLK.value)
    name_zh: Mapped[str] = mapped_column(String(256))
    name_en: Mapped[str] = mapped_column(String(256))
    category: Mapped[str] = mapped_column(String(32), default="MISC")
    spec: Mapped[str] = mapped_column(String(256), default="")
    unit: Mapped[str] = mapped_column(String(16), default="pcs")
    min_stock: Mapped[int] = mapped_column(Integer, default=0)
    location_id: Mapped[int | None] = mapped_column(
        Integer, ForeignKey("locations.id"), nullable=True
    )
    note: Mapped[str] = mapped_column(Text, default="")
    active: Mapped[bool] = mapped_column(default=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=func.now()
    )

    location: Mapped["Location | None"] = relationship()
    stock: Mapped["Stock | None"] = relationship(
        back_populates="item", uselist=False, cascade="all, delete-orphan"
    )


class Stock(Base):
    __tablename__ = "stock"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    item_id: Mapped[int] = mapped_column(
        Integer, ForeignKey("items.id"), unique=True
    )
    quantity: Mapped[int] = mapped_column(Integer, default=0)

    item: Mapped["Item"] = relationship(back_populates="stock")


class Transaction(Base):
    __tablename__ = "transactions"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    type: Mapped[str] = mapped_column(String(16))
    item_id: Mapped[int] = mapped_column(Integer, ForeignKey("items.id"))
    quantity: Mapped[int] = mapped_column(Integer, default=1)
    operator_id: Mapped[int | None] = mapped_column(
        Integer, ForeignKey("operators.id"), nullable=True
    )
    note: Mapped[str] = mapped_column(Text, default="")
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=func.now(), index=True
    )

    item: Mapped["Item"] = relationship()
    operator: Mapped["Operator | None"] = relationship()


class AppSetting(Base):
    __tablename__ = "app_settings"

    key: Mapped[str] = mapped_column(String(64), primary_key=True)
    value: Mapped[str] = mapped_column(Text, default="")
