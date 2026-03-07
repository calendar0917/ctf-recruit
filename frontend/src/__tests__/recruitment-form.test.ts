import { describe, expect, it } from "vitest";
import {
  getMissingRecruitmentFields,
  normalizeRecruitmentFormValue,
} from "@/lib/recruitment-form";

describe("recruitment form validation", () => {
  it("returns all required field labels when empty", () => {
    const missing = getMissingRecruitmentFields({
      name: "",
      school: "",
      grade: "",
      direction: "",
      contact: "",
      bio: "",
    });

    expect(missing).toEqual(["姓名", "学校", "年级", "方向", "联系方式", "个人简介"]);
  });

  it("normalizes values and passes required validation for submit flow", () => {
    const normalized = normalizeRecruitmentFormValue({
      name: " Alice ",
      school: " Test University ",
      grade: " 大二 ",
      direction: " web ",
      contact: " alice@example.com ",
      bio: " loves ctf ",
    });

    expect(normalized).toEqual({
      name: "Alice",
      school: "Test University",
      grade: "大二",
      direction: "web",
      contact: "alice@example.com",
      bio: "loves ctf",
    });

    expect(getMissingRecruitmentFields(normalized)).toEqual([]);
  });
});
