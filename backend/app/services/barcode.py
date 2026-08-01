from sqlalchemy.orm import Session

from app.models import Item, Program, TrackMode


def _prefix(program: str, track_mode: str) -> str:
    prog = program if program in (Program.FRC.value, Program.FTC.value) else "FIRST"
    mode = "SNP" if track_mode == TrackMode.SNP.value else "BLK"
    return f"{prog}-{mode}"


def _max_serial(db: Session, prefix: str) -> int:
    pattern = f"{prefix}-%"
    rows = db.query(Item.barcode).filter(Item.barcode.like(pattern)).all()
    max_num = 0
    for (code,) in rows:
        try:
            max_num = max(max_num, int(code.rsplit("-", 1)[-1]))
        except ValueError:
            continue
    return max_num


def next_barcode(db: Session, program: str, track_mode: str) -> str:
    prefix = _prefix(program, track_mode)
    return f"{prefix}-{_max_serial(db, prefix) + 1:05d}"
